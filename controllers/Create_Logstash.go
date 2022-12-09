package controllers

import (
	"context"
	"fmt"

	apiv1 "github.com/huisebug/logfile-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *LogFileReconciler) LogstashCreteConfigMap(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {

	var logstashconf, logstashyml string
	customizelog := logger.WithValues("func", "LogstashCreteConfigMap")

	// 此处必须配置，已达到去掉默认logstash.yml中的xpack.monitoring.elasticsearch.hosts，不然后续会在es设置kibana用户密码后认证es时会提示401
	logstashyml = `
http.host: "0.0.0.0"
`

	switch logfile.Spec.ProgrammeNum {
	case 3:
		logstashconf = fmt.Sprintf(`
input {
  # 配置接收Filebeat数据源，监听端口为5044
  # Filebeat的output.logstash地址保持跟这里一致
  beats {
    port => 5044
  }
}

output {
  # 将数据导入到ES中
  elasticsearch {
    hosts => ["http://elasticsearch.logfile-operator-system:9200"]
    index => "logfile-operator-logstash-%%{+yyyy.MM.dd}"

    # 需要不断新建索引和进行写入索引，不然会提示：
    # only write ops with an op_type of create are allowed in data streams
    action => "create"
     #es的用户名和密码
    user => "elastic"
    password => %s
  }
  stdout {
    codec => rubydebug
  }  
}	
`, logfile.Spec.KIBANA_PASSWORD)

	case 4:
		logstashconf = fmt.Sprintf(`
input {
  # 配置接收Filebeat数据源，监听端口为5044
  # Filebeat的output.logstash地址保持跟这里一致
  beats {
    port => 5044
  }
}
output {
  #将数据导入到ES中
  elasticsearch {  
    index => "logfile-operator-logstash-%%{+yyyy.MM.dd}"
    # 处理因kafka转换过的日志内容
    codec => line { format => "%%{message}"}
    # 需要不断新建索引和进行写入索引，不然会提示：
    # only write ops with an op_type of create are allowed in data streams
    action => "create"
    #es的用户名和密码
    user => "elastic"
    password => %s
    #使用https的es8集群配置
    hosts => ["https://elasticsearch-master.logfile-operator-system:9200"]
    ssl => true
    #crt证书的所在路径
    cacert => '/usr/share/elasticsearch/config/certs/ca.crt'
  }
  stdout {
    codec => rubydebug
  }
}	
`, logfile.Spec.KIBANA_PASSWORD)

	case 5:
		logstashconf = fmt.Sprintf(`
input {
  kafka {
    #kafka地址
    bootstrap_servers => "kafka.logfile-operator-system:9092"
    # kafka主题
    topics => "kafka_log"
    # 消费者线程数
    consumer_threads => 1
    # 当 Kafka 中没有初始偏移量或偏移量超出范围时该怎么办
    auto_offset_reset => "latest"
    # 处理因kafka转换过的日志内容
    codec => "json"
  }
}

output {
  # 将数据导入到ES中
  elasticsearch {
    hosts => ["http://elasticsearch.logfile-operator-system:9200"]
    index => "logfile-operator-kafka-logstash-%%{+yyyy.MM.dd}"
    # 处理因kafka转换过的日志内容
    codec => line { format => "%%{message}"}
    # 需要不断新建索引和进行写入索引，不然会提示：
    # only write ops with an op_type of create are allowed in data streams
    action => "create"
     #es的用户名和密码
    user => "elastic"
    password => %s
  }
  stdout {
    codec => rubydebug
  }
}	
`, logfile.Spec.KIBANA_PASSWORD)

	case 6:
		logstashconf = fmt.Sprintf(`
input {
  kafka {
    #kafka地址
    bootstrap_servers => "kafka-cluster-headless.logfile-operator-system:9092"
    # kafka主题
    topics => "kafka_log"
    # 消费者线程数
    consumer_threads => 1
    # 当 Kafka 中没有初始偏移量或偏移量超出范围时该怎么办
    auto_offset_reset => "latest"
    # 处理因kafka转换过的日志内容
    codec => "json"
  }
}

output {
  #将数据导入到ES中
  elasticsearch {  
    index => "logfile-operator-kafka-cluster-logstash-%%{+yyyy.MM.dd}"
    # 处理因kafka转换过的日志内容
    codec => line { format => "%%{message}"}
    # 需要不断新建索引和进行写入索引，不然会提示：
    # only write ops with an op_type of create are allowed in data streams
    action => "create"
    #es的用户名和密码
    user => "elastic"
    password => %s
    #使用https的es8集群配置
    hosts => ["https://elasticsearch-master.logfile-operator-system:9200"]
    ssl => true
    #crt证书的所在路径
    cacert => '/usr/share/elasticsearch/config/certs/ca.crt'
  }
  stdout {
    codec => rubydebug
  }
}	
`, logfile.Spec.KIBANA_PASSWORD)

	}

	ymlmap := make(map[string]string)
	confmap := make(map[string]string)
	ymlmap["logstash.yml"] = logstashyml
	confmap["logstash.conf"] = logstashconf

	ymlmeta := meta.DeepCopy()
	ymlmeta.Name += "yml"

	ymlconfigmap := &corev1.ConfigMap{
		ObjectMeta: *ymlmeta,
		Data:       ymlmap,
	}
	confmeta := meta.DeepCopy()
	confmeta.Name += "conf"

	confconfigmap := &corev1.ConfigMap{
		ObjectMeta: *confmeta,
		Data:       confmap,
	}

	// 级联删除
	customizelog.Info("set configmap reference")
	if err := controllerutil.SetControllerReference(logfile, ymlconfigmap, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	customizelog.Info("set configmap reference")
	if err := controllerutil.SetControllerReference(logfile, confconfigmap, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	// 新建
	if err := r.Create(ctx, ymlconfigmap); err != nil {
		return err
	}
	if err := r.Create(ctx, confconfigmap); err != nil {
		return err
	}
	customizelog.Info("create configmap success", "name", typesname.String())

	return nil
}

func (r *LogFileReconciler) LogstashCreteDeployment(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {
	customizelog := logger.WithValues("func", "LogstashCreteDeployment")

	env := []corev1.EnvVar{
		{
			Name:  "I18N_LOCALE",
			Value: "zh-CN",
		},
	}

	volume := []corev1.Volume{
		{
			Name: meta.Name + "yml",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: pointer.Int32(420),
					LocalObjectReference: corev1.LocalObjectReference{
						Name: meta.Name + "yml",
					},
				},
			},
		},
		{
			Name: meta.Name + "conf",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: pointer.Int32(420),
					LocalObjectReference: corev1.LocalObjectReference{
						Name: meta.Name + "conf",
					},
				},
			},
		},
	}

	volumemount := []corev1.VolumeMount{
		{
			Name:      meta.Name + "yml",
			MountPath: "/usr/share/logstash/config/logstash.yml",
			SubPath:   "logstash.yml",
		},
		{
			Name:      meta.Name + "conf",
			MountPath: "/usr/share/logstash/pipeline/logstash.conf",
			SubPath:   "logstash.conf",
		},
	}

	if logfile.Spec.ProgrammeNum == 4 || logfile.Spec.ProgrammeNum == 6 {
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
							Name:            "logstash",
							Image:           "registry.cn-hangzhou.aliyuncs.com/huisebug/logfile-operator:logstash-8.5.0",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env:             env,
							Ports: []corev1.ContainerPort{
								{
									Name:          "logstash",
									Protocol:      corev1.Protocol("TCP"),
									ContainerPort: int32(5044),
								},
							},

							VolumeMounts: volumemount,
						},
					},
					Volumes: volume,
				},
			},
		},
	}

	// 当logstash作为kafka消费者时候，不会运行5044端口服务
	if logfile.Spec.ProgrammeNum == 3 || logfile.Spec.ProgrammeNum == 4 {
		livenssprobe := &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.IntOrString{
						Type:   1,
						IntVal: 0,
						StrVal: "logstash",
					},
				},
			},
			PeriodSeconds:       *pointer.Int32(120),
			InitialDelaySeconds: *pointer.Int32(120),
			TimeoutSeconds:      *pointer.Int32(120),
		}
		readinessprobe := &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.IntOrString{
						Type:   1,
						IntVal: 0,
						StrVal: "logstash",
					},
				},
			},
			PeriodSeconds:       *pointer.Int32(120),
			InitialDelaySeconds: *pointer.Int32(30),
			TimeoutSeconds:      *pointer.Int32(120),
		}

		for index, _ := range deployment.Spec.Template.Spec.Containers {
			if deployment.Spec.Template.Spec.Containers[index].Name == "logstash" {
				deployment.Spec.Template.Spec.Containers[index].LivenessProbe = livenssprobe
				deployment.Spec.Template.Spec.Containers[index].ReadinessProbe = readinessprobe
			}
		}

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
func (r *LogFileReconciler) LogstashCreteService(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {
	customizelog := logger.WithValues("func", "LogstashCreteService")

	service := &corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{

			Selector:                 labels,
			PublishNotReadyAddresses: false,
			Ports: []corev1.ServicePort{
				{
					Name:     "logstash",
					Port:     int32(5044),
					Protocol: corev1.ProtocolTCP,
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
