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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const annotationRegExpString = "logfile\\.huisebug\\.org\\/[a-zA-Z\\.]+"

type LogFileAnnotation struct {
	annotationRegExp *regexp.Regexp
}

type AutoGenerated struct {
	Filebeatinputs []Filebeatinputs `yaml:"filebeat.inputs"`
}
type Filebeatinputs struct {
	Type  string   `yaml:"type"`
	Paths []string `yaml:"paths"`
}

func FilebeatfileGen(logfilepaths []string) string {

	t := AutoGenerated{
		Filebeatinputs: []Filebeatinputs{
			Filebeatinputs{
				Type:  "log",
				Paths: logfilepaths,
			},
		},
	}

	d, err := yaml.Marshal(&t)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return string(d)
}

func (l *LogFileAnnotation) NewLogFileAnnotation() *LogFileAnnotation {
	r, _ := regexp.Compile(annotationRegExpString)
	return &LogFileAnnotation{
		annotationRegExp: r,
	}
}

func (l *LogFileAnnotation) filterAnnotations(annotations map[string]string) map[string]string {
	Annotations := make(map[string]string)
	for key, value := range annotations {
		if l.annotationRegExp.MatchString(key) {
			Annotations[key] = value
		}
	}
	return Annotations
}

const annotationDomainSeparator = "/"
const annotationSubDomainSeparator = "."

func parseMetrics(annotations map[string]string, podName string) []string {

	var metrics []string
	// 循环注解从正则过滤后的注解
	for metricKey, metricValue := range annotations {
		// 以/为分隔符来拆分，判断key名是否长度为2,如果不是则不符合要求
		keys := strings.Split(metricKey, annotationDomainSeparator)
		if len(keys) != 2 {
			logrus.Errorf("Metric annotation for %v  is invalid: %v", podName, metricKey)
			return metrics
		}
		// 以.为分隔符来拆分索引0的域名，判断域名是否长度小于2,如果小于则不符合域名规范
		metricSubDomains := strings.Split(keys[0], annotationSubDomainSeparator)
		if len(metricSubDomains) < 2 {
			logrus.Errorf("Metric annotation for  %v is invalid: %v", podName, metricKey)
			return metrics
		}
		// 对域名的主机位进行判断，是否是想要的主机位进行开头
		switch metricSubDomains[0] {
		case "logfile":
			// 以,为分隔符拆分多个日志文件路径
			metricValueslice := strings.Split(metricValue, ",")
			metrics = append(metrics, metricValueslice...)
		}

	}

	return metrics
}

// PodSideCarMutate mutate Pods
type PodSidecarMutate struct {
	Client  client.Client
	decoder *admission.Decoder
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

func NewPodSideCarMutate(c client.Client) admission.Handler {
	return &PodSidecarMutate{Client: c}
}

func Createk8sClientSet() *kubernetes.Clientset {
	// 创建集群内配置
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println(err)
	}
	// 创建客户端
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println(err)
	}
	return clientset
}

func Createk8sDiscoveryClient() *discovery.DiscoveryClient {
	// 创建集群内配置
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println(err)
	}
	// 创建客户端
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		fmt.Println(err)
	}
	return discoveryClient
}

func Formatbase64string(secretdatavalue []byte) string {
	// k8s存放的是2次base64编码后的，所以要转2次,第二次中存在了=号，要进行特别解码
	one := base64.StdEncoding.EncodeToString(secretdatavalue)
	// fmt.Println("one", one)

	two, _ := base64.RawStdEncoding.DecodeString(one)
	// fmt.Println(string(two))
	return string(two)
}

// PodSideCarMutate admits a pod if a specific annotation exists.
func (v *PodSidecarMutate) Handle(ctx context.Context, req admission.Request) admission.Response {
	// TODO

	pod := &corev1.Pod{}

	err := v.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	// fmt.Printf("pod: %+v\n", pod)
	fmt.Printf("pod元数据名称: %+v\n", pod.ObjectMeta.GenerateName)
	fmt.Printf("pod的所有标签信息: %+v\n", pod.Labels)
	fmt.Printf("pod的所有注释信息: %+v\n", pod.Annotations)

	lfa := new(LogFileAnnotation)
	lfa = lfa.NewLogFileAnnotation()

	// 获取过滤后的注解map
	Annotations := lfa.filterAnnotations(pod.Annotations)
	// 获取符合域名规则的注解
	logfilepaths := parseMetrics(Annotations, pod.Name)

	// 使用ClientSet
	clientset := Createk8sClientSet()
	// 获取configmap
	configmap, configmaperr := clientset.CoreV1().ConfigMaps("logfile-operator-system").Get(context.TODO(), "filebeat-sidecar", metav1.GetOptions{})
	// fmt.Printf("configmap: %+v\n", configmap)

	configmapstatus := func(configmap *corev1.ConfigMap, configmaperr error) bool {
		if errors.IsNotFound(configmaperr) {
			return false
		}
		if _, ok := configmap.Data["filebeat.yml"]; ok {
			if _, ok := configmap.Data["programmenumber"]; ok {
				return true
			}
		}
		return false

	}(configmap, configmaperr)

	switch {
	case pod.Labels["pod-admission-webhook-injection"] == "false":
		Tips := fmt.Sprintf("Namespace: %s; Pod: %s; 存在Label: pod-admission-webhook-injection: \"false\"; 跳过注入sidecar", pod.Namespace, pod.ObjectMeta.Name)
		log.Println(Tips)
	case len(logfilepaths) == 0:
		Tips := fmt.Sprintf("Namespace: %s; Pod: %s; 未在注释中声明: logfile.huisebug.org字段: \"容器日志文件路径1,容器日志文件路径2\"; 跳过注入sidecar", pod.Namespace, pod.ObjectMeta.Name)
		log.Println(Tips)
	case !configmapstatus:
		log.Println("未查询到: logfile-operator-system; configmap: filebeat-sidecar 中键值为:filebeat.yml和programmenumber的数据; 跳过注入sidecar")
	default:

		confdir := corev1.Volume{
			Name: "confdir",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, confdir)
		// 获取日志文件所在的文件目录
		logfilepath_dirs := func(logfilepaths []string) []string {
			tmplist := []string{}
			for _, logfilepath := range logfilepaths {
				logfilepath_dir, _ := filepath.Split(logfilepath)
				tmplist = append(tmplist, logfilepath_dir)
			}
			return tmplist
		}(logfilepaths)

		// 日志文件所在的文件目录去重
		logfilepath_dirs = func(slc []string) []string {
			result := []string{}
			tempMap := map[string]byte{} // 存放不重复主键
			for _, e := range slc {
				l := len(tempMap)
				tempMap[e] = 0
				if len(tempMap) != l { // 加入map后，map长度变化，则元素不重复
					result = append(result, e)
				}
			}
			return result
		}(logfilepath_dirs)

		// 临时给需要注入的服务也增加filebeat的配置文件目录
		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{
				Name:      "confdir",
				MountPath: "/etc/filebeat/",
			},
		)

		// 生成filebeat配置文件所需的执行命令,配置由日志文件路径和configmap中配置输出位置组成
		commandline := fmt.Sprintf(`
echo '
%s
%s
' > /etc/filebeat/filebeat.yml
`, FilebeatfileGen(logfilepaths), configmap.Data["filebeat.yml"])

		// 利用initcontainer生成filebeat的配置文件
		sidecarinitcontainer := corev1.Container{
			Name:            "genfilebeatyml",
			Image:           "registry.cn-hangzhou.aliyuncs.com/huisebug/logfile-operator:filebeat-8.5.0",
			ImagePullPolicy: corev1.PullIfNotPresent,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "confdir",
					MountPath: "/etc/filebeat/",
				},
			},
			Command: []string{
				"/bin/bash",
				"-c",
			},
			Args: []string{
				commandline,
			},
		}

		// sidecar注入容器的信息
		Privileged := true
		sidecarcontainer := corev1.Container{
			SecurityContext: &corev1.SecurityContext{
				Privileged: &Privileged,
			},
			Name:            "filebeat",
			Image:           "registry.cn-hangzhou.aliyuncs.com/huisebug/logfile-operator:filebeat-8.5.0",
			ImagePullPolicy: corev1.PullIfNotPresent,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "confdir",
					MountPath: "/etc/filebeat/",
				},
			},
			Args: []string{"-e", "-c", "/etc/filebeat/filebeat.yml"},
		}

		// 循环所有的日志文件目录，将文件目录都创建EmptyDir卷声明和挂载到将要sidecar注入的filebeat容器中
		for index, logfilepath_dir := range logfilepath_dirs {

			// 判断此路径是否已经进行挂载,如果已经挂载就返回其挂载信息
			Duplicate_mount_point := func(l string, pod *corev1.Pod) []corev1.VolumeMount {
				VolumeMounts := []corev1.VolumeMount{}
				for _, container := range pod.Spec.Containers {

					for _, VolumeMount := range container.VolumeMounts {
						// 判断此路径是否已经是挂载路径
						// 判断是否以 / 结尾
						if strings.HasSuffix(VolumeMount.MountPath, "/") {
							if l == VolumeMount.MountPath {
								VolumeMounts = append(VolumeMounts, VolumeMount)
							}
						} else {
							if l == VolumeMount.MountPath+"/" {
								VolumeMounts = append(VolumeMounts, VolumeMount)
							}
						}

					}

				}
				return VolumeMounts

			}(logfilepath_dir, pod)

			log.Printf("日志文件目录：%s; 重复挂载点列表Duplicate_mount_point: %+v\n", logfilepath_dir, Duplicate_mount_point)

			// 如果挂载重复挂载点列表长度为0，说明没有重复挂载点，就需要创建EmptyDir
			if len(Duplicate_mount_point) == 0 {
				EmptyDir := corev1.Volume{
					Name: "logfile-operator-" + strconv.Itoa(index),
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				}
				pod.Spec.Volumes = append(pod.Spec.Volumes, EmptyDir)
				VolumeMount := corev1.VolumeMount{
					Name:      "logfile-operator-" + strconv.Itoa(index),
					MountPath: logfilepath_dir,
				}
				// 往将要创建pod中默认的容器列表都进行卷挂载
				for c_index, _ := range pod.Spec.Containers {
					pod.Spec.Containers[c_index].VolumeMounts = append(pod.Spec.Containers[c_index].VolumeMounts, VolumeMount)
				}
				// 往sidecar容器进行卷挂载
				sidecarcontainer.VolumeMounts = append(sidecarcontainer.VolumeMounts, VolumeMount)
			} else {
				// 往sidecar容器进行卷挂载
				sidecarcontainer.VolumeMounts = append(sidecarcontainer.VolumeMounts, Duplicate_mount_point...)
			}

		}

		// 获取方案的序号
		programmenumber := configmap.Data["programmenumber"]
		if programmenumber == "2" {
			secret, _ := clientset.CoreV1().Secrets("logfile-operator-system").Get(context.TODO(), "elasticsearch-master-certs", metav1.GetOptions{})
			// fmt.Printf("secret: %+v\n", secret)

			// 生成方案2时，filebeat需要使用es8集群的https证书
			elasticsearchcerts := corev1.Volume{
				Name: "elasticsearch-master-certs",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			}
			pod.Spec.Volumes = append(pod.Spec.Volumes, elasticsearchcerts)

			commandline := fmt.Sprintf(`
echo '
%s
' > /usr/share/elasticsearch/config/certs/tls.crt \
&& echo '
%s
' > /usr/share/elasticsearch/config/certs/tls.key \
&& echo '
%s
' > /usr/share/elasticsearch/config/certs/ca.crt
`, Formatbase64string(secret.Data["tls.crt"]), Formatbase64string(secret.Data["tls.key"]), Formatbase64string(secret.Data["ca.crt"]))

			// fmt.Println(Formatbase64string(secret.Data["ca.crt"]))

			// 利用initcontainer生成filebeat的配置文件
			eshttpscrtinitcontainer := corev1.Container{
				Name:            "genesclusterhttps",
				Image:           "debian:stretch-slim",
				ImagePullPolicy: corev1.PullIfNotPresent,
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "elasticsearch-master-certs",
						MountPath: "/usr/share/elasticsearch/config/certs",
					},
				},
				Command: []string{
					"/bin/bash",
					"-c",
				},
				Args: []string{
					commandline,
				},
			}
			// 将新增的容器加入到pod中
			pod.Spec.InitContainers = append(pod.Spec.InitContainers, eshttpscrtinitcontainer)
			// 给filebeat容器挂载上elasticsearch的https证书
			sidecarcontainer.VolumeMounts = append(sidecarcontainer.VolumeMounts, corev1.VolumeMount{
				Name:      "elasticsearch-master-certs",
				MountPath: "/usr/share/elasticsearch/config/certs",
			})
		}

		// 将新增的容器加入到pod中
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, sidecarinitcontainer)
		pod.Spec.Containers = append(pod.Spec.Containers, sidecarcontainer)
	}

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// PodSideCarMutate 实现 admission.DecoderInjector。
// 将自动注入解码器。
// 注入解码器
func (v *PodSidecarMutate) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
