package controllers

import (
	"context"
	"fmt"
	"strconv"

	apiv1 "github.com/huisebug/logfile-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *LogFileReconciler) FilebeatCreteConfigMap(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {
	var filebeatyml string
	customizelog := logger.WithValues("func", "FilebeatCreteConfigMap")
	switch logfile.Spec.ProgrammeNum {
	case 1:
		filebeatyml = fmt.Sprintf(`
output.elasticsearch:
  index: "logfile-operator-filebeat-%%{+yyyy.MM.dd}"
  hosts: ['http://elasticsearch.logfile-operator-system:9200']
  username: elastic
  password: %s
#自定义索引名
setup.template.name: "logfile-operator-filebeat"
setup.template.pattern: "logfile-operator-filebeat-*"

http.enabled: true
http.host: 0.0.0.0
`, logfile.Spec.ELASTIC_PASSWORD)

	case 2:
		filebeatyml = fmt.Sprintf(`
output.elasticsearch:
  index: "logfile-operator-filebeat-%%{+yyyy.MM.dd}"
  hosts: ['https://elasticsearch-master.logfile-operator-system:9200']
  protocol: https
  ssl.certificate_authorities: ["/usr/share/elasticsearch/config/certs/ca.crt"]
  username: elastic
  password: %s
#自定义索引名
setup.template.name: "logfile-operator-filebeat"
setup.template.pattern: "logfile-operator-filebeat-*"

http.enabled: true
http.host: 0.0.0.0
`, logfile.Spec.ELASTIC_PASSWORD)

	case 3, 4:
		filebeatyml = `
output.logstash:
  # 使用logstash
  hosts: ['logstash.logfile-operator-system:5044']

http.enabled: true
http.host: 0.0.0.0
`

	case 5:
		filebeatyml = `
output.kafka:
  # 使用kafka
  hosts: ['kafka.logfile-operator-system:9092']
  # 主题
  topic: kafka_log
  # 大于max_message_bytes将被丢弃的事件
  max_message_bytes: 1000000

http.enabled: true
http.host: 0.0.0.0
`

	case 6:
		filebeatyml = `
output.kafka:
  # 使用kafka集群
  hosts: ['kafka-cluster-headless.logfile-operator-system:9092']
  # 主题
  topic: kafka_log
  # 大于max_message_bytes将被丢弃的事件
  max_message_bytes: 1000000

http.enabled: true
http.host: 0.0.0.0
`

	}
	tmpmap := make(map[string]string)
	tmpmap["filebeat.yml"] = filebeatyml
	// 传递方案序号，让sidecar可以获取到
	tmpmap["programmenumber"] = strconv.Itoa(logfile.Spec.ProgrammeNum)

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
