package controllers

import (
	"context"
	"log"

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

func (r *LogFileReconciler) KafkaClusterCreteServiceAccount(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {
	var AutomountServiceAccountToken = true
	customizelog := logger.WithValues("func", "KafkaClusterCreteServiceAccount")

	sa := &corev1.ServiceAccount{
		ObjectMeta:                   meta,
		AutomountServiceAccountToken: &AutomountServiceAccountToken,
	}

	// 级联删除
	customizelog.Info("set sa reference")
	if err := controllerutil.SetControllerReference(logfile, sa, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	// 新建
	if err := r.Create(ctx, sa); err != nil {
		return err
	}

	customizelog.Info("create sa success", "name", typesname.String())

	return nil
}

func (r *LogFileReconciler) KafkaClusterCreteConfigMap(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {

	customizelog := logger.WithValues("func", "KafkaClusterCreteConfigMap")

	kafkamap := make(map[string]string)

	setupsh := []byte(`
#!/bin/bash
ID="${MY_POD_NAME#"kafka-cluster-"}"
if [[ -f "/bitnami/kafka/data/meta.properties" ]]; then
    export KAFKA_CFG_BROKER_ID="$(grep "broker.id" "/bitnami/kafka/data/meta.properties" | awk -F '=' '{print $2}')"
else
    export KAFKA_CFG_BROKER_ID="$((ID + 0))"
fi
# Configure zookeeper client
exec /entrypoint.sh /run.sh
`)

	kafkamap["setup.sh"] = string(setupsh)

	kafkameta := meta.DeepCopy()
	kafkameta.Name += "-scripts"

	kafkaconfigmap := &corev1.ConfigMap{
		ObjectMeta: *kafkameta,
		Data:       kafkamap,
	}

	// 级联删除
	if err := controllerutil.SetControllerReference(logfile, kafkaconfigmap, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	// 新建
	if err := r.Create(ctx, kafkaconfigmap); err != nil {
		return err
	}

	customizelog.Info("create kafkaconfigmap success", "name", typesname.String())

	return nil
}

func (r *LogFileReconciler) KafkaClusterCreteService(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {

	customizelog := logger.WithValues("func", "KafkaClusterCreteService")

	kafkaheadlessmeta := meta.DeepCopy()
	kafkaheadlessmeta.Name = kafkaheadlessmeta.Name + "-headless"

	kafkaserviceheadless := &corev1.Service{
		ObjectMeta: *kafkaheadlessmeta,
		Spec: corev1.ServiceSpec{
			Type:                     corev1.ServiceTypeClusterIP,
			ClusterIP:                "None",
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					Name:     "tcp-client",
					Port:     int32(9092),
					Protocol: corev1.ProtocolTCP,
					TargetPort: intstr.IntOrString{
						Type:   1,
						IntVal: 0,
						StrVal: "kafka-client",
					},
				},
				{
					Name:     "tcp-internal",
					Port:     int32(9093),
					Protocol: corev1.ProtocolTCP,
					TargetPort: intstr.IntOrString{
						Type:   1,
						IntVal: 0,
						StrVal: "kafka-internal",
					},
				},
			},
			Selector: labels,
		},
	}

	kafkaservice := &corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Type:            corev1.ServiceTypeClusterIP,
			SessionAffinity: corev1.ServiceAffinityNone,
			Selector:        labels,
			Ports: []corev1.ServicePort{
				{
					Name:     "tcp-client",
					Port:     int32(9092),
					Protocol: corev1.ProtocolTCP,
					TargetPort: intstr.IntOrString{
						Type:   1,
						IntVal: 0,
						StrVal: "kafka-client",
					},
				},
			},
		},
	}

	// 级联删除service
	customizelog.Info("set serviceheadless reference")
	if err := controllerutil.SetControllerReference(logfile, kafkaserviceheadless, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	if err := controllerutil.SetControllerReference(logfile, kafkaservice, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	// 新建service
	if err := r.Create(ctx, kafkaserviceheadless); err != nil {
		return err
	}
	if err := r.Create(ctx, kafkaservice); err != nil {
		return err
	}

	customizelog.Info("create kafkaserviceheadless and kafkaservice success", "name", typesname.String())

	return nil
}

func (r *LogFileReconciler) KafkaClusterCreteStatefulSet(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {
	customizelog := logger.WithValues("func", "KafkaClusterCreteStatefulSet")

	var RunAsNonRoot = true

	var AllowPrivilegeEscalation = false

	// 申请storageclass的大小
	log.Printf("%+v\n", logfile.Spec.ResourceStorage)
	var resourceStorage, _ = resource.ParseQuantity(logfile.Spec.ResourceStorage.Kafka)

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
			Replicas:            pointer.Int32Ptr(3),
			PodManagementPolicy: appsv1.PodManagementPolicyType("Parallel"),
			Selector:            metav1.SetAsLabelSelector(labels),
			ServiceName:         meta.Name + "-headless",
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.StatefulSetUpdateStrategyType("RollingUpdate"),
			},

			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					HostNetwork:        false,
					HostIPC:            false,
					ServiceAccountName: meta.Name,
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "app",
												Operator: "In",
												Values:   []string{meta.Name},
											},
										},
									},
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: pointer.Int64(1001),
					},
					TerminationGracePeriodSeconds: pointer.Int64(120),

					Volumes: []corev1.Volume{
						{
							Name: "scripts",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: meta.Name + "-scripts",
									},
									DefaultMode: pointer.Int32(0755),
								},
							},
						},
						{
							Name: "logs",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},

					Containers: []corev1.Container{
						{
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:                pointer.Int64(1001),
								RunAsNonRoot:             &RunAsNonRoot,
								AllowPrivilegeEscalation: &AllowPrivilegeEscalation,
							},
							Name:            "kafka",
							Image:           "registry.cn-hangzhou.aliyuncs.com/huisebug/logfile-operator:kafka-3.3",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"bash", "-c", "/scripts/setup.sh"},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.IntOrString{
											Type:   1,
											IntVal: 0,
											StrVal: "kafka-client",
										},
									},
								},
								FailureThreshold:    *pointer.Int32(3),
								InitialDelaySeconds: *pointer.Int32(10),
								PeriodSeconds:       *pointer.Int32(10),
								SuccessThreshold:    *pointer.Int32(1),
								TimeoutSeconds:      *pointer.Int32(5),
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.IntOrString{
											Type:   1,
											IntVal: 0,
											StrVal: "kafka-client",
										},
									},
								},
								FailureThreshold:    *pointer.Int32(6),
								InitialDelaySeconds: *pointer.Int32(5),
								PeriodSeconds:       *pointer.Int32(10),
								SuccessThreshold:    *pointer.Int32(1),
								TimeoutSeconds:      *pointer.Int32(5),
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "kafka-client",
									Protocol:      corev1.Protocol("TCP"),
									ContainerPort: int32(9092),
								},
								{
									Name:          "kafka-internal",
									Protocol:      corev1.Protocol("TCP"),
									ContainerPort: int32(9093),
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits:   corev1.ResourceList{},
								Requests: corev1.ResourceList{},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "BITNAMI_DEBUG",
									Value: "false",
								},
								{
									Name: "MY_POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name: "MY_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name:  "KAFKA_CFG_ZOOKEEPER_CONNECT",
									Value: "kafka-cluster-zookeeper",
								},
								{
									Name:  "KAFKA_INTER_BROKER_LISTENER_NAME",
									Value: "INTERNAL",
								},
								{
									Name:  "KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP",
									Value: "INTERNAL:PLAINTEXT,CLIENT:PLAINTEXT",
								},
								{
									Name:  "KAFKA_CFG_LISTENERS",
									Value: "INTERNAL://:9093,CLIENT://:9092",
								},
								{
									Name:  "KAFKA_CFG_ADVERTISED_LISTENERS",
									Value: "INTERNAL://$(MY_POD_NAME).kafka-cluster-headless.logfile-operator-system.svc.cluster.local:9093,CLIENT://$(MY_POD_NAME).kafka-cluster-headless.logfile-operator-system.svc.cluster.local:9092",
								},
								{
									Name:  "ALLOW_PLAINTEXT_LISTENER",
									Value: "yes",
								},
								{
									Name:  "KAFKA_ZOOKEEPER_PROTOCOL",
									Value: "PLAINTEXT",
								},
								{
									Name:  "KAFKA_VOLUME_DIR",
									Value: "/bitnami/kafka",
								},
								{
									Name:  "KAFKA_LOG_DIR",
									Value: "/opt/bitnami/kafka/logs",
								},
								{
									Name:  "KAFKA_CFG_DELETE_TOPIC_ENABLE",
									Value: "false",
								},
								{
									Name:  "KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE",
									Value: "true",
								},
								{
									Name:  "KAFKA_HEAP_OPTS",
									Value: "-Xmx1024m -Xms1024m",
								},
								{
									Name:  "KAFKA_CFG_LOG_FLUSH_INTERVAL_MESSAGES",
									Value: "10000",
								},
								{
									Name:  "KAFKA_CFG_LOG_FLUSH_INTERVAL_MS",
									Value: "1000",
								},
								{
									Name:  "KAFKA_CFG_LOG_RETENTION_BYTES",
									Value: "1073741824",
								},
								{
									Name:  "KAFKA_CFG_LOG_RETENTION_CHECK_INTERVAL_MS",
									Value: "300000",
								},
								{
									Name:  "KAFKA_CFG_LOG_RETENTION_HOURS",
									Value: "168",
								},
								{
									Name:  "KAFKA_CFG_MESSAGE_MAX_BYTES",
									Value: "1000012",
								},
								{
									Name:  "KAFKA_CFG_LOG_SEGMENT_BYTES",
									Value: "1073741824",
								},
								{
									Name:  "KAFKA_CFG_LOG_DIRS",
									Value: "/bitnami/kafka/data",
								},
								{
									Name:  "KAFKA_CFG_DEFAULT_REPLICATION_FACTOR",
									Value: "1",
								},
								{
									Name:  "KAFKA_CFG_OFFSETS_TOPIC_REPLICATION_FACTOR",
									Value: "1",
								},
								{
									Name:  "KAFKA_CFG_TRANSACTION_STATE_LOG_REPLICATION_FACTOR",
									Value: "1",
								},
								{
									Name:  "KAFKA_CFG_TRANSACTION_STATE_LOG_MIN_ISR",
									Value: "1",
								},
								{
									Name:  "KAFKA_CFG_NUM_IO_THREADS",
									Value: "8",
								},
								{
									Name:  "KAFKA_CFG_NUM_NETWORK_THREADS",
									Value: "3",
								},
								{
									Name:  "KAFKA_CFG_NUM_PARTITIONS",
									Value: "1",
								},
								{
									Name:  "KAFKA_CFG_NUM_RECOVERY_THREADS_PER_DATA_DIR",
									Value: "1",
								},
								{
									Name:  "KAFKA_CFG_SOCKET_RECEIVE_BUFFER_BYTES",
									Value: "102400",
								},
								{
									Name:  "KAFKA_CFG_SOCKET_REQUEST_MAX_BYTES",
									Value: "104857600",
								},
								{
									Name:  "KAFKA_CFG_SOCKET_SEND_BUFFER_BYTES",
									Value: "102400",
								},
								{
									Name:  "KAFKA_CFG_ZOOKEEPER_CONNECTION_TIMEOUT_MS",
									Value: "6000",
								},
								{
									Name:  "KAFKA_CFG_AUTHORIZER_CLASS_NAME",
									Value: "",
								},
								{
									Name:  "KAFKA_CFG_ALLOW_EVERYONE_IF_NO_ACL_FOUND",
									Value: "true",
								},
								{
									Name:  "KAFKA_CFG_SUPER_USERS",
									Value: "User:admin",
								},
							},

							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      meta.Name,
									MountPath: "/bitnami/kafka",
								},
								{
									Name:      "logs",
									MountPath: "/opt/bitnami/kafka/logs",
								},
								{
									Name:      "scripts",
									MountPath: "/scripts/setup.sh",
									SubPath:   "setup.sh",
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
