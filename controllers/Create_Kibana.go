package controllers

import (
	"context"
	"fmt"
	"log"

	apiv1 "github.com/huisebug/logfile-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *LogFileReconciler) KibanaCreteConfigMap(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {

	var kibanayml string
	customizelog := logger.WithValues("func", "KibanaCreteConfigMap")

	switch logfile.Spec.ProgrammeNum {
	case 1, 3, 5:
		kibanayml = fmt.Sprintf(`
server.name: kibana
server.host: 0.0.0.0
elasticsearch.hosts: [ "http://elasticsearch.logfile-operator-system:9200" ]
monitoring.ui.container.elasticsearch.enabled: true
elasticsearch.username: kibana_system
elasticsearch.password: %s
`, logfile.Spec.KIBANA_PASSWORD)
	case 2, 4, 6:
		kibanayml = fmt.Sprintf(`
server.name: kibana
server.host: 0.0.0.0
elasticsearch.hosts: [ "https://elasticsearch-master.logfile-operator-system:9200" ]
monitoring.ui.container.elasticsearch.enabled: true
elasticsearch.username: kibana_system
elasticsearch.password: %s
`, logfile.Spec.KIBANA_PASSWORD)
	}

	tmpmap := make(map[string]string)
	tmpmap["kibana.yml"] = kibanayml

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

func (r *LogFileReconciler) KibanaCreteDeployment(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {
	customizelog := logger.WithValues("func", "KibanaCreteDeployment")

	env := []corev1.EnvVar{
		{
			Name:  "I18N_LOCALE",
			Value: "zh-CN",
		},
	}

	volume := []corev1.Volume{
		{
			Name: "conf",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: pointer.Int32(420),
					LocalObjectReference: corev1.LocalObjectReference{
						Name: meta.Name,
					},
				},
			},
		},
	}

	volumemount := []corev1.VolumeMount{
		{
			Name:      "conf",
			MountPath: "/usr/share/kibana/config",
		},
	}

	if logfile.Spec.ProgrammeNum == 2 || logfile.Spec.ProgrammeNum == 4 || logfile.Spec.ProgrammeNum == 6 {
		env = append(env,
			corev1.EnvVar{
				Name:  "ELASTICSEARCH_SSL_CERTIFICATEAUTHORITIES",
				Value: "/usr/share/elasticsearch/config/certs/ca.crt",
			},
		)
		volumemount = append(volumemount,
			corev1.VolumeMount{
				Name:      "elasticsearch-master-certs",
				MountPath: "/usr/share/elasticsearch/config/certs/",
				ReadOnly:  true,
			},
		)
		volume = append(volume,
			corev1.Volume{
				Name: "elasticsearch-master-certs",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "elasticsearch-master-certs",
					},
				},
			},
		)
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: meta,
		Spec: appsv1.DeploymentSpec{

			Replicas: pointer.Int32Ptr(1),
			Selector: metav1.SetAsLabelSelector(labels),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "kibana",
							Image:           "registry.cn-hangzhou.aliyuncs.com/huisebug/logfile-operator:kibana-8.5.0",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env:             env,
							Ports: []corev1.ContainerPort{
								{
									Name:          "kibana",
									Protocol:      corev1.Protocol("TCP"),
									ContainerPort: int32(5601),
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.IntOrString{
											Type:   1,
											IntVal: 0,
											StrVal: "kibana",
										},
									},
								},
								PeriodSeconds:       *pointer.Int32(120),
								InitialDelaySeconds: *pointer.Int32(60),
								TimeoutSeconds:      *pointer.Int32(120),
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.IntOrString{
											Type:   1,
											IntVal: 0,
											StrVal: "kibana",
										},
									},
								},
								PeriodSeconds:       *pointer.Int32(120),
								InitialDelaySeconds: *pointer.Int32(60),
								TimeoutSeconds:      *pointer.Int32(120),
							},
							VolumeMounts: volumemount,
						},
					},
					Volumes: volume,
				},
			},
		},
	}

	// 级联删除deployment
	customizelog.Info("set deployment reference")
	if err := controllerutil.SetControllerReference(logfile, deployment, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	// 新建deployment
	if err := r.Create(ctx, deployment); err != nil {
		return err
	}
	customizelog.Info("create deployment success", "name", typesname.String())
	return nil
}
func (r *LogFileReconciler) KibanaCreteService(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {
	customizelog := logger.WithValues("func", "KibanaCreteService")
	// 暴露NodePort
	log.Printf("%+v\n", logfile.Spec.NodePortS)
	service := &corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Type:                     corev1.ServiceTypeNodePort,
			Selector:                 labels,
			PublishNotReadyAddresses: false,
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     int32(5601),
					Protocol: corev1.ProtocolTCP,
					NodePort: int32(logfile.Spec.NodePortS.Kibana),
				},
			},
		},
	}

	// 级联删除service
	if err := controllerutil.SetControllerReference(logfile, service, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	// 新建service
	if err := r.Create(ctx, service); err != nil {
		return err
	}

	customizelog.Info("create  service success", "name", typesname.String())

	return nil
}
