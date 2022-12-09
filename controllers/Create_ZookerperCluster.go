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

func (r *LogFileReconciler) ZookerperClusterCreteConfigMap(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {

	customizelog := logger.WithValues("func", "ZookerperClusterCreteConfigMap")

	zookeepermap := make(map[string]string)

	setupsh := []byte(`
#!/bin/bash
if [[ -f "/bitnami/zookeeper/data/myid" ]]; then
    export ZOO_SERVER_ID="$(cat /bitnami/zookeeper/data/myid)"
else
    HOSTNAME="$(hostname -s)"
    if [[ $HOSTNAME =~ (.*)-([0-9]+)$ ]]; then
        ORD=${BASH_REMATCH[2]}
        export ZOO_SERVER_ID="$((ORD + 1 ))"
    else
        echo "Failed to get index from hostname $HOST"
        exit 1
    fi
fi
exec /entrypoint.sh /run.sh
`)

	zookeepermap["setup.sh"] = string(setupsh)

	zookeepermeta := meta.DeepCopy()
	zookeepermeta.Name += "-scripts"

	zookeeperconfigmap := &corev1.ConfigMap{
		ObjectMeta: *zookeepermeta,
		Data:       zookeepermap,
	}

	// 级联删除
	customizelog.Info("set configmap reference")
	if err := controllerutil.SetControllerReference(logfile, zookeeperconfigmap, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}

	// 新建
	if err := r.Create(ctx, zookeeperconfigmap); err != nil {
		return err
	}

	customizelog.Info("create zookeeperconfigmap success", "name", typesname.String())

	return nil
}

func (r *LogFileReconciler) ZookerperClusterCreteService(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {

	customizelog := logger.WithValues("func", "ZookerperClusterCreteService")

	zookeeperheadlessmeta := meta.DeepCopy()
	zookeeperheadlessmeta.Name = zookeeperheadlessmeta.Name + "-headless"

	zookeeperserviceheadless := &corev1.Service{
		ObjectMeta: *zookeeperheadlessmeta,
		Spec: corev1.ServiceSpec{
			ClusterIP:                "None",
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					Name:     "tcp-client",
					Port:     int32(2181),
					Protocol: corev1.ProtocolTCP,
					TargetPort: intstr.IntOrString{
						Type:   1,
						IntVal: 0,
						StrVal: "client",
					},
				},
				{
					Name:     "tcp-follower",
					Port:     int32(2888),
					Protocol: corev1.ProtocolTCP,
					TargetPort: intstr.IntOrString{
						Type:   1,
						IntVal: 0,
						StrVal: "follower",
					},
				},
				{
					Name:     "tcp-election",
					Port:     int32(3888),
					Protocol: corev1.ProtocolTCP,
					TargetPort: intstr.IntOrString{
						Type:   1,
						IntVal: 0,
						StrVal: "election",
					},
				},
			},
			Selector: labels,
		},
	}

	zookeeperservice := &corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Type:            corev1.ServiceTypeClusterIP,
			SessionAffinity: corev1.ServiceAffinityNone,
			Selector:        labels,
			Ports: []corev1.ServicePort{
				{
					Name:     "tcp-client",
					Port:     int32(2181),
					Protocol: corev1.ProtocolTCP,
					TargetPort: intstr.IntOrString{
						Type:   1,
						IntVal: 0,
						StrVal: "client",
					},
				},
				{
					Name:     "tcp-follower",
					Port:     int32(2888),
					Protocol: corev1.ProtocolTCP,
					TargetPort: intstr.IntOrString{
						Type:   1,
						IntVal: 0,
						StrVal: "follower",
					},
				},
				{
					Name:     "tcp-election",
					Port:     int32(3888),
					Protocol: corev1.ProtocolTCP,
					TargetPort: intstr.IntOrString{
						Type:   1,
						IntVal: 0,
						StrVal: "election",
					},
				},
			},
		},
	}

	// 级联删除service
	customizelog.Info("set serviceheadless reference")
	if err := controllerutil.SetControllerReference(logfile, zookeeperserviceheadless, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	if err := controllerutil.SetControllerReference(logfile, zookeeperservice, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	// 新建service
	if err := r.Create(ctx, zookeeperserviceheadless); err != nil {
		return err
	}
	if err := r.Create(ctx, zookeeperservice); err != nil {
		return err
	}

	customizelog.Info("create zookeeperserviceheadless and zookeeperservice success", "name", typesname.String())

	return nil
}

func (r *LogFileReconciler) ZookerperClusterCreteStatefulSet(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {
	customizelog := logger.WithValues("func", "ZookerperClusterCreteStatefulSet")

	var RunAsNonRoot = true

	var AllowPrivilegeEscalation = false

	// 申请storageclass的大小
	log.Printf("%+v\n", logfile.Spec.ResourceStorage)
	var resourceStorage, _ = resource.ParseQuantity(logfile.Spec.ResourceStorage.Zookeeper)

	ProbeCommand := []string{"/bin/bash", "-c", "echo ruok | timeout 2 nc -w 2 localhost 2181 | grep imok"}

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
					ServiceAccountName: "default",
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
					},

					Containers: []corev1.Container{
						{
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:                pointer.Int64(1001),
								RunAsNonRoot:             &RunAsNonRoot,
								AllowPrivilegeEscalation: &AllowPrivilegeEscalation,
							},
							Name:            "zookeeper",
							Image:           "registry.cn-hangzhou.aliyuncs.com/huisebug/logfile-operator:zookeeper-3.8",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"bash", "-c", "/scripts/setup.sh"},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: ProbeCommand,
									},
								},

								FailureThreshold:    *pointer.Int32(6),
								InitialDelaySeconds: *pointer.Int32(30),
								PeriodSeconds:       *pointer.Int32(10),
								SuccessThreshold:    *pointer.Int32(1),
								TimeoutSeconds:      *pointer.Int32(5),
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: ProbeCommand,
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
									Name:          "client",
									Protocol:      corev1.Protocol("TCP"),
									ContainerPort: int32(2181),
								},
								{
									Name:          "follower",
									Protocol:      corev1.Protocol("TCP"),
									ContainerPort: int32(2888),
								},
								{
									Name:          "election",
									Protocol:      corev1.Protocol("TCP"),
									ContainerPort: int32(3888),
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("250m"),
									"memory": resource.MustParse("256Mi"),
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "TZ",
									Value: "Asia/Shanghai",
								},
								{
									Name:  "BITNAMI_DEBUG",
									Value: "false",
								},
								{
									Name:  "ZOO_DATA_LOG_DIR",
									Value: "",
								},
								{
									Name:  "ZOO_PORT_NUMBER",
									Value: "2181",
								},
								{
									Name:  "ZOO_TICK_TIME",
									Value: "2000",
								},
								{
									Name:  "ZOO_INIT_LIMIT",
									Value: "10",
								},
								{
									Name:  "ZOO_SYNC_LIMIT",
									Value: "5",
								},
								{
									Name:  "ZOO_PRE_ALLOC_SIZE",
									Value: "65536",
								},
								{
									Name:  "ZOO_SNAPCOUNT",
									Value: "100000",
								},
								{
									Name:  "ZOO_MAX_CLIENT_CNXNS",
									Value: "60",
								},
								{
									Name:  "ZOO_4LW_COMMANDS_WHITELIST",
									Value: "srvr, mntr, ruok",
								},
								{
									Name:  "ZOO_LISTEN_ALLIPS_ENABLED",
									Value: "no",
								},
								{
									Name:  "ZOO_AUTOPURGE_INTERVAL",
									Value: "0",
								},
								{
									Name:  "ZOO_AUTOPURGE_RETAIN_COUNT",
									Value: "3",
								},
								{
									Name:  "ZOO_MAX_SESSION_TIMEOUT",
									Value: "40000",
								},
								{
									Name:  "ZOO_SERVERS",
									Value: "kafka-cluster-zookeeper-0.kafka-cluster-zookeeper-headless.logfile-operator-system.svc.cluster.local:2888:3888::1 kafka-cluster-zookeeper-1.kafka-cluster-zookeeper-headless.logfile-operator-system.svc.cluster.local:2888:3888::2 kafka-cluster-zookeeper-2.kafka-cluster-zookeeper-headless.logfile-operator-system.svc.cluster.local:2888:3888::3",
								},
								{
									Name:  "ZOO_ENABLE_AUTH",
									Value: "no",
								},
								{
									Name:  "ZOO_ENABLE_QUORUM_AUTH",
									Value: "no",
								},
								{
									Name:  "ZOO_HEAP_SIZE",
									Value: "1024",
								},
								{
									Name:  "ZOO_LOG_LEVEL",
									Value: "ERROR",
								},
								{
									Name:  "ALLOW_ANONYMOUS_LOGIN",
									Value: "yes",
								},
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath:  "metadata.name",
											APIVersion: "v1",
										},
									},
								},
							},

							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      meta.Name,
									MountPath: "/bitnami/zookeeper",
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
