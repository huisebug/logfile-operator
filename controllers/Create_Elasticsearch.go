package controllers

import (
	"context"
	"log"

	apiv1 "github.com/huisebug/logfile-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *LogFileReconciler) ElasticsearchCreteConfigMap(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {

	customizelog := logger.WithValues("func", "ElasticsearchCreteConfigMap")

	tmpmap := make(map[string]string)
	tmpmap["elasticsearch.yml"] = `
cluster.name: "docker-cluster"
network.host: 0.0.0.0
xpack.license.self_generated.type: basic
xpack.security.enabled: true
`

	configmap := &corev1.ConfigMap{
		ObjectMeta: meta,
		Data:       tmpmap,
	}

	// 级联删除
	customizelog.Info("set configmap reference")
	if err := controllerutil.SetControllerReference(logfile, configmap, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	// 新建
	if err := r.Create(ctx, configmap); err != nil {
		return err
	}

	customizelog.Info("create configmap success", "name", typesname.String())

	return nil
}

func (r *LogFileReconciler) ElasticsearchCreteStatefulSet(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {

	customizelog := logger.WithValues("func", "ElasticsearchCreteStatefulSet")

	// 申请storageclass的大小
	log.Printf("%+v\n", logfile.Spec.ResourceStorage)
	var resourceStorage, _ = resource.ParseQuantity(logfile.Spec.ResourceStorage.Elasticsearch)

	statefulset := &appsv1.StatefulSet{
		ObjectMeta: meta,
		Spec: appsv1.StatefulSetSpec{
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: meta.Name,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadOnlyMany,
						},
						Resources: corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceStorage: resourceStorage,
							},
						},

						StorageClassName: &logfile.Spec.StorageClassName,
					},
				},
			},
			Replicas:    pointer.Int32Ptr(1),
			Selector:    metav1.SetAsLabelSelector(labels),
			ServiceName: meta.Name + "-headless",
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "elasticsearch",
							Image:           "registry.cn-hangzhou.aliyuncs.com/huisebug/logfile-operator:elasticsearch-8.5.0",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env: []corev1.EnvVar{
								{
									Name:  "TZ",
									Value: "Asia/Shanghai",
								},
								{
									Name:  "discovery.type",
									Value: "single-node",
								},
								{
									Name:  "ES_JAVA_OPTS",
									Value: "-Xms512m -Xmx512m",
								},
								{
									Name:  "ELASTIC_PASSWORD",
									Value: logfile.Spec.ELASTIC_PASSWORD,
								},
								{
									Name:  "KIBANA_PASSWORD",
									Value: logfile.Spec.KIBANA_PASSWORD,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "elasticsearch",
									Protocol:      corev1.Protocol("TCP"),
									ContainerPort: int32(9200),
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.IntOrString{
											Type:   1,
											IntVal: 0,
											StrVal: "elasticsearch",
										},
									},
								},
								PeriodSeconds:       *pointer.Int32(120),
								InitialDelaySeconds: *pointer.Int32(120),
								TimeoutSeconds:      *pointer.Int32(120),
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.IntOrString{
											Type:   1,
											IntVal: 0,
											StrVal: "elasticsearch",
										},
									},
								},
								PeriodSeconds:       *pointer.Int32(120),
								InitialDelaySeconds: *pointer.Int32(60),
								TimeoutSeconds:      *pointer.Int32(120),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "elasticsearch",
									MountPath: "/usr/share/elasticsearch/data",
								},
								{
									Name:      "conf",
									MountPath: "/usr/share/elasticsearch/config/elasticsearch.yml",
									SubPath:   "elasticsearch.yml",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "conf",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: meta.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// 级联删除statefulset
	customizelog.Info("set statefulset reference")
	if err := controllerutil.SetControllerReference(logfile, statefulset, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	// 新建statefulset
	if err := r.Create(ctx, statefulset); err != nil {
		return err
	}

	customizelog.Info("create statefulset success", "name", typesname.String())

	return nil
}
func (r *LogFileReconciler) ElasticsearchKibanaUserCreteJob(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {

	customizelog := logger.WithValues("func", "ElasticsearchKibanaUserCreteJob")
	job := &batchv1.Job{
		ObjectMeta: meta,
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "set-kibana-user-password",
							Image:           "registry.cn-hangzhou.aliyuncs.com/huisebug/logfile-operator:elasticsearch-8.5.0",
							ImagePullPolicy: "IfNotPresent",
							Env: []corev1.EnvVar{
								{
									Name:  "ELASTIC_PASSWORD",
									Value: logfile.Spec.ELASTIC_PASSWORD,
								},
								{
									Name:  "KIBANA_PASSWORD",
									Value: logfile.Spec.KIBANA_PASSWORD,
								},
							},
							Command: []string{
								"/bin/bash",
								"-c",
							},
							Args: []string{
								`until curl -s -X POST -u "elastic:${ELASTIC_PASSWORD}" -H "Content-Type: application/json" http://elasticsearch.logfile-operator-system:9200/_security/user/kibana_system/_password -d "{\"password\":\"${KIBANA_PASSWORD}\"}" | grep -q "^{}"; do sleep 10; done;`,
							},
						},
					},
				},
			},
		},
	}

	// 级联删除job
	customizelog.Info("set job reference")
	if err := controllerutil.SetControllerReference(logfile, job, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}

	// 新建job
	if err := r.Create(ctx, job); err != nil {
		return err
	}

	customizelog.Info("create job success", "name", typesname.String())

	return nil
}

func (r *LogFileReconciler) ElasticsearchCreteService(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {

	customizelog := logger.WithValues("func", "ElasticsearchCreteStatefulSet")

	// 暴露NodePort
	log.Printf("%+v\n", logfile.Spec.NodePortS)
	headlessmeta := meta.DeepCopy()
	headlessmeta.Name = headlessmeta.Name + "-headless"

	serviceheadless := &corev1.Service{
		ObjectMeta: *headlessmeta,
		Spec: corev1.ServiceSpec{
			ClusterIP:                "None",
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     int32(9200),
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name:     "transport",
					Port:     int32(9300),
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: labels,
		},
	}

	service := &corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Type:                     corev1.ServiceTypeNodePort,
			Selector:                 labels,
			PublishNotReadyAddresses: false,
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     int32(9200),
					Protocol: corev1.ProtocolTCP,
					NodePort: int32(logfile.Spec.NodePortS.Elasticsearch),
				},
				{
					Name:     "transport",
					Port:     int32(9300),
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	// 级联删除service
	customizelog.Info("set serviceheadless reference")
	if err := controllerutil.SetControllerReference(logfile, serviceheadless, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	if err := controllerutil.SetControllerReference(logfile, service, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	// 新建service
	if err := r.Create(ctx, serviceheadless); err != nil {
		return err
	}
	if err := r.Create(ctx, service); err != nil {
		return err
	}

	customizelog.Info("create serviceheadless and service success", "name", typesname.String())

	return nil
}
