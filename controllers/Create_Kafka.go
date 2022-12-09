package controllers

import (
	"context"
	"log"
	"strconv"
	"strings"

	apiv1 "github.com/huisebug/logfile-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func FormatStorageint(storage string) int {
	if strings.HasSuffix(storage, "Gi") {
		desStr1 := strings.Split(storage, "G")
		num, err := strconv.Atoi(desStr1[0])
		if err != nil {
			return 0
		}
		return num
	}
	return 0
}

func (r *LogFileReconciler) KafkaCreteStatefulSet(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {
	customizelog := logger.WithValues("func", "KafkaCreteStatefulSet")

	// 申请storageclass的大小
	log.Printf("%+v\n", logfile.Spec.ResourceStorage)
	// 当kafka进行单实例部署时，将zookeeper和kafka申请的存储空间进行总和申请
	Sum := FormatStorageint(logfile.Spec.ResourceStorage.Zookeeper) + FormatStorageint(logfile.Spec.ResourceStorage.Kafka)
	var resourceStorage, _ = resource.ParseQuantity(strconv.Itoa(Sum))

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
							Name:            "zookeeper",
							Image:           "registry.cn-hangzhou.aliyuncs.com/huisebug/logfile-operator:zookeeper-3.8",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env: []corev1.EnvVar{
								{
									Name:  "ALLOW_ANONYMOUS_LOGIN",
									Value: "yes",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "zookeeper",
									Protocol:      corev1.Protocol("TCP"),
									ContainerPort: int32(2181),
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.IntOrString{
											Type:   1,
											IntVal: 0,
											StrVal: "zookeeper",
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
											StrVal: "zookeeper",
										},
									},
								},
								PeriodSeconds:       *pointer.Int32(120),
								InitialDelaySeconds: *pointer.Int32(60),
								TimeoutSeconds:      *pointer.Int32(120),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/bitnami/zookeeper",
									Name:      meta.Name,
									SubPath:   "zookeeper",
								},
							},
						},
						{
							Name:            "kafka",
							Image:           "registry.cn-hangzhou.aliyuncs.com/huisebug/logfile-operator:kafka-3.3",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env: []corev1.EnvVar{
								{
									Name:  "KAFKA_BROKER_ID",
									Value: "1",
								},
								{
									Name:  "KAFKA_CFG_LISTENERS",
									Value: "PLAINTEXT://:9092",
								},
								{
									Name:  "KAFKA_CFG_ADVERTISED_LISTENERS",
									Value: "PLAINTEXT://kafka.logfile-operator-system:9092",
								},
								{
									Name:  "KAFKA_CFG_ZOOKEEPER_CONNECT",
									Value: "127.0.0.1:2181",
								},
								{
									Name:  "ALLOW_PLAINTEXT_LISTENER",
									Value: "yes",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "kafka",
									Protocol:      corev1.Protocol("TCP"),
									ContainerPort: int32(9092),
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.IntOrString{
											Type:   1,
											IntVal: 0,
											StrVal: "kafka",
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
											StrVal: "kafka",
										},
									},
								},
								PeriodSeconds:       *pointer.Int32(120),
								InitialDelaySeconds: *pointer.Int32(60),
								TimeoutSeconds:      *pointer.Int32(120),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/bitnami/kafka",
									Name:      meta.Name,
									SubPath:   "kafka",
								},
							},
						},
					},
				},
			},
		},
	}

	// 级联删除
	customizelog.Info("set statefulset reference")
	if err := controllerutil.SetControllerReference(logfile, statefulset, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	// 新建
	if err := r.Create(ctx, statefulset); err != nil {
		return err
	}
	customizelog.Info("create statefulset success", "name", typesname.String())
	return nil
}
func (r *LogFileReconciler) KafkaCreteService(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {
	customizelog := logger.WithValues("func", "KafkaCreteService")

	headlessmeta := meta.DeepCopy()
	headlessmeta.Name = headlessmeta.Name + "-headless"

	serviceheadless := &corev1.Service{
		ObjectMeta: *headlessmeta,
		Spec: corev1.ServiceSpec{
			ClusterIP:                "None",
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					Name:     "kafka",
					Port:     int32(9092),
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: labels,
		},
	}
	service := &corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{

			Selector:                 labels,
			PublishNotReadyAddresses: false,
			Ports: []corev1.ServicePort{
				{
					Name:     "kafka",
					Port:     int32(9092),
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	// 级联删除service
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

	customizelog.Info("create  serviceheadless and service success", "name", typesname.String())

	return nil
}
