/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"reflect"
	"time"

	apiv1 "github.com/huisebug/logfile-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// LogFileReconciler reconciles a LogFile object
type LogFileReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var Status = apiv1.LogFileStatus{
	Status: "Active",
}

//+kubebuilder:rbac:groups=api.huisebug.org,resources=logfiles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=api.huisebug.org,resources=logfiles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=api.huisebug.org,resources=logfiles/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
//可以往其他namespace写入event
//+kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete

func (r *LogFileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	defer utilruntime.HandleCrash()
	customizelog := logger.WithValues("func", req.NamespacedName)
	// your logic here
	customizelog.Info("1.start logfile reconcile")

	// 实例化数据结构
	logfile := &apiv1.LogFile{}

	// 查询自定义资源是否存在req.NamespacedName的值
	if err := r.Get(ctx, req.NamespacedName, logfile); err != nil {
		if errors.IsNotFound(err) {
			customizelog.Info("2.1 instance not found, 可能已移除", "func", "Reconcile")
			// 包reconcile的结构体
			return reconcile.Result{}, nil
		}
		customizelog.Error(err, "2.2 error")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 如果处在删除中直接跳过
	if logfile.DeletionTimestamp != nil {
		customizelog.Info("logfile in deleting", "name", req.String())
		return ctrl.Result{}, nil
	}
	// 如果处在活跃状态说明已经运行一个方案了，直接跳过
	if logfile.Status == Status {
		customizelog.Info("logfile in Already active", "name", req.String())
		return ctrl.Result{}, nil
	}
	// 运行
	if err := r.Run(ctx, logfile); err != nil {
		customizelog.Error(err, "failed to Run logfile", "name", req.String())
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *LogFileReconciler) Run(ctx context.Context, logfile *apiv1.LogFile) error {
	var err error
	customizelog := logger.WithValues("func", "Run")

	logfile = logfile.DeepCopy()
	logfilename := types.NamespacedName{
		Namespace: logfile.Namespace,
		Name:      logfile.Name,
	}
	owner := []metav1.OwnerReference{
		{
			APIVersion:         logfile.APIVersion,
			Kind:               logfile.Kind,
			Name:               logfile.Name,
			Controller:         pointer.BoolPtr(true),
			BlockOwnerDeletion: pointer.BoolPtr(true),
			UID:                logfile.UID,
		},
	}
	labels := map[string]string{
		"logfile-operator": logfile.Name,
	}

	meta := metav1.ObjectMeta{
		OwnerReferences: owner,
	}
	customizelog.Info("logfile operatra 采用方案序号", logfile.Spec.ProgrammeNum)

	// 创建filebeat输出位置configmap

	filebeatmeta := meta.DeepCopy()
	filebeatmeta.Name = "filebeat-sidecar"
	filebeatmeta.Namespace = "logfile-operator-system"
	labels["app"] = filebeatmeta.Name
	filebeatmeta.Labels = labels
	if err = r.FilebeatCreteConfigMap(ctx, logfile, logfilename, *filebeatmeta, labels); err != nil {
		return err
	}

	// 创建kafka对应的方案序号
	switch logfile.Spec.ProgrammeNum {
	case 5:
		// 定义统一的部署类型名称
		kafkameta := meta.DeepCopy()
		kafkameta.Name = "kafka"
		kafkameta.Namespace = "logfile-operator-system"
		labels["app"] = kafkameta.Name
		kafkameta.Labels = labels
		if err = r.KafkaCreteService(ctx, logfile, logfilename, *kafkameta, labels); err != nil {
			return err
		}
		if err = r.KafkaCreteStatefulSet(ctx, logfile, logfilename, *kafkameta, labels); err != nil {
			return err
		}
	case 6:
		// 定义统一的部署类型名称
		zookeepermeta := meta.DeepCopy()
		zookeepermeta.Name = "kafka-cluster-zookeeper"
		zookeepermeta.Namespace = "logfile-operator-system"
		labels["app"] = zookeepermeta.Name
		zookeepermeta.Labels = labels
		if err = r.ZookerperClusterCreteConfigMap(ctx, logfile, logfilename, *zookeepermeta, labels); err != nil {
			return err
		}
		if err = r.ZookerperClusterCreteService(ctx, logfile, logfilename, *zookeepermeta, labels); err != nil {
			return err
		}
		if err = r.ZookerperClusterCreteStatefulSet(ctx, logfile, logfilename, *zookeepermeta, labels); err != nil {
			return err
		}

		kafkameta := meta.DeepCopy()
		kafkameta.Name = "kafka-cluster"
		kafkameta.Namespace = "logfile-operator-system"
		labels["app"] = kafkameta.Name
		kafkameta.Labels = labels
		if err = r.KafkaClusterCreteConfigMap(ctx, logfile, logfilename, *kafkameta, labels); err != nil {
			return err
		}
		if err = r.KafkaClusterCreteService(ctx, logfile, logfilename, *kafkameta, labels); err != nil {
			return err
		}
		if err = r.KafkaClusterCreteServiceAccount(ctx, logfile, logfilename, *kafkameta, labels); err != nil {
			return err
		}
		if err = r.KafkaClusterCreteStatefulSet(ctx, logfile, logfilename, *kafkameta, labels); err != nil {
			return err
		}
	}

	// 创建logstash对应的方案序号
	switch logfile.Spec.ProgrammeNum {
	case 3, 4, 5, 6:
		// 定义统一的部署类型名称
		logstashmeta := meta.DeepCopy()
		logstashmeta.Name = "logstash"
		logstashmeta.Namespace = "logfile-operator-system"
		labels["app"] = logstashmeta.Name
		logstashmeta.Labels = labels
		if err = r.LogstashCreteConfigMap(ctx, logfile, logfilename, *logstashmeta, labels); err != nil {
			return err
		}
		if err = r.LogstashCreteDeployment(ctx, logfile, logfilename, *logstashmeta, labels); err != nil {
			return err
		}
		if err = r.LogstashCreteService(ctx, logfile, logfilename, *logstashmeta, labels); err != nil {
			return err
		}
	}

	// 创建elasticsearch对应的方案序号
	switch logfile.Spec.ProgrammeNum {
	case 1, 3, 5:
		// 定义统一的部署类型名称
		elasticesearchmeta := meta.DeepCopy()
		elasticesearchmeta.Name = "elasticsearch"
		elasticesearchmeta.Namespace = "logfile-operator-system"
		labels["app"] = elasticesearchmeta.Name
		elasticesearchmeta.Labels = labels
		if err = r.ElasticsearchCreteConfigMap(ctx, logfile, logfilename, *elasticesearchmeta, labels); err != nil {
			return err
		}
		if err = r.ElasticsearchCreteService(ctx, logfile, logfilename, *elasticesearchmeta, labels); err != nil {
			return err
		}
		if err = r.ElasticsearchCreteStatefulSet(ctx, logfile, logfilename, *elasticesearchmeta, labels); err != nil {
			return err
		}
		customizelog.Info("等待Elasticsearch创建80秒后再创建设置kibana用户密码Job")

		time.Sleep(time.Duration(80) * time.Second)
		elasticesearchmeta.Name += "-set-kibana-password"
		if err = r.ElasticsearchKibanaUserCreteJob(ctx, logfile, logfilename, *elasticesearchmeta, labels); err != nil {
			return err
		}

	case 2, 4, 6:
		// 定义统一的部署类型名称
		elasticesearchmeta := meta.DeepCopy()
		elasticesearchmeta.Name = "elasticsearch-master"
		elasticesearchmeta.Namespace = "logfile-operator-system"
		labels["app"] = elasticesearchmeta.Name
		elasticesearchmeta.Labels = labels

		if err = r.ElasticsearchClusterCretePodDisruptionBudget(ctx, logfile, logfilename, *elasticesearchmeta, labels); err != nil {
			return err
		}
		if err = r.ElasticsearchClusterCreteSecret(ctx, logfile, logfilename, *elasticesearchmeta, labels); err != nil {
			return err
		}
		if err = r.ElasticsearchClusterCreteService(ctx, logfile, logfilename, *elasticesearchmeta, labels); err != nil {
			return err
		}
		if err = r.ElasticsearchClusterCreteStatefulSet(ctx, logfile, logfilename, *elasticesearchmeta, labels); err != nil {
			return err
		}
		customizelog.Info("等待Elasticsearch集群创建80秒后再创建设置kibana用户密码Job")
		time.Sleep(time.Duration(80) * time.Second)
		elasticesearchmeta.Name += "-set-kibana-password"
		if err = r.ElasticsearchClusterKibanaUserCreteJob(ctx, logfile, logfilename, *elasticesearchmeta, labels); err != nil {
			return err
		}

	}

	// 等待KibanaUser创建成功
	customizelog.Info("等待elasticsearch-set-kibana-password Job设置kibana用户密码后20秒再创建kibana")
	time.Sleep(time.Duration(20) * time.Second)
	kibanameta := meta.DeepCopy()
	kibanameta.Name = "kibana"
	kibanameta.Namespace = "logfile-operator-system"
	labels["app"] = kibanameta.Name
	kibanameta.Labels = labels
	if err = r.KibanaCreteConfigMap(ctx, logfile, logfilename, *kibanameta, labels); err != nil {
		return err
	}
	if err = r.KibanaCreteService(ctx, logfile, logfilename, *kibanameta, labels); err != nil {
		return err
	}
	if err = r.KibanaCreteDeployment(ctx, logfile, logfilename, *kibanameta, labels); err != nil {
		return err
	}

	// 更新状态

	if !reflect.DeepEqual(logfile.Status, Status) {
		logfile.Status = Status
		customizelog.Info("update logfile status", "name", logfilename.String())
		return r.Client.Status().Update(ctx, logfile)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
// 使用的是 Builder 模式，NewControllerManagerBy 和 For 方法都是给 Builder 传参，最重要的是最后一个方法 Complete
func (r *LogFileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 3,
		}).
		For(&apiv1.LogFile{}).
		//下面的单独监听会导致多次触发Reconcile的执行
		// Owns(&appsv1.Deployment{}).
		// Owns(&corev1.Service{}).
		Complete(r)
}
