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
	// ???????????????????????????????????????
	for metricKey, metricValue := range annotations {
		// ???/??????????????????????????????key??????????????????2,??????????????????????????????
		keys := strings.Split(metricKey, annotationDomainSeparator)
		if len(keys) != 2 {
			logrus.Errorf("Metric annotation for %v  is invalid: %v", podName, metricKey)
			return metrics
		}
		// ???.???????????????????????????0??????????????????????????????????????????2,????????????????????????????????????
		metricSubDomains := strings.Split(keys[0], annotationSubDomainSeparator)
		if len(metricSubDomains) < 2 {
			logrus.Errorf("Metric annotation for  %v is invalid: %v", podName, metricKey)
			return metrics
		}
		// ???????????????????????????????????????????????????????????????????????????
		switch metricSubDomains[0] {
		case "logfile":
			// ???,??????????????????????????????????????????
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
	// ?????????????????????
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println(err)
	}
	// ???????????????
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println(err)
	}
	return clientset
}

func Createk8sDiscoveryClient() *discovery.DiscoveryClient {
	// ?????????????????????
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println(err)
	}
	// ???????????????
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		fmt.Println(err)
	}
	return discoveryClient
}

func Formatbase64string(secretdatavalue []byte) string {
	// k8s????????????2???base64???????????????????????????2???,?????????????????????=???????????????????????????
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
	fmt.Printf("pod???????????????: %+v\n", pod.ObjectMeta.GenerateName)
	fmt.Printf("pod?????????????????????: %+v\n", pod.Labels)
	fmt.Printf("pod?????????????????????: %+v\n", pod.Annotations)

	lfa := new(LogFileAnnotation)
	lfa = lfa.NewLogFileAnnotation()

	// ????????????????????????map
	Annotations := lfa.filterAnnotations(pod.Annotations)
	// ?????????????????????????????????
	logfilepaths := parseMetrics(Annotations, pod.Name)

	// ??????ClientSet
	clientset := Createk8sClientSet()
	// ??????configmap
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
		Tips := fmt.Sprintf("Namespace: %s; Pod: %s; ??????Label: pod-admission-webhook-injection: \"false\"; ????????????sidecar", pod.Namespace, pod.ObjectMeta.Name)
		log.Println(Tips)
	case len(logfilepaths) == 0:
		Tips := fmt.Sprintf("Namespace: %s; Pod: %s; ?????????????????????: logfile.huisebug.org??????: \"????????????????????????1,????????????????????????2\"; ????????????sidecar", pod.Namespace, pod.ObjectMeta.Name)
		log.Println(Tips)
	case !configmapstatus:
		log.Println("????????????: logfile-operator-system; configmap: filebeat-sidecar ????????????:filebeat.yml???programmenumber?????????; ????????????sidecar")
	default:

		confdir := corev1.Volume{
			Name: "confdir",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, confdir)
		// ???????????????????????????????????????
		logfilepath_dirs := func(logfilepaths []string) []string {
			tmplist := []string{}
			for _, logfilepath := range logfilepaths {
				logfilepath_dir, _ := filepath.Split(logfilepath)
				tmplist = append(tmplist, logfilepath_dir)
			}
			return tmplist
		}(logfilepaths)

		// ???????????????????????????????????????
		logfilepath_dirs = func(slc []string) []string {
			result := []string{}
			tempMap := map[string]byte{} // ?????????????????????
			for _, e := range slc {
				l := len(tempMap)
				tempMap[e] = 0
				if len(tempMap) != l { // ??????map??????map?????????????????????????????????
					result = append(result, e)
				}
			}
			return result
		}(logfilepath_dirs)

		// ???????????????????????????????????????filebeat?????????????????????
		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{
				Name:      "confdir",
				MountPath: "/etc/filebeat/",
			},
		)

		// ??????filebeat?????????????????????????????????,??????????????????????????????configmap???????????????????????????
		commandline := fmt.Sprintf(`
echo '
%s
%s
' > /etc/filebeat/filebeat.yml
`, FilebeatfileGen(logfilepaths), configmap.Data["filebeat.yml"])

		// ??????initcontainer??????filebeat???????????????
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

		// sidecar?????????????????????
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

		// ????????????????????????????????????????????????????????????EmptyDir???????????????????????????sidecar?????????filebeat?????????
		for index, logfilepath_dir := range logfilepath_dirs {

			// ???????????????????????????????????????,??????????????????????????????????????????
			Duplicate_mount_point := func(l string, pod *corev1.Pod) []corev1.VolumeMount {
				VolumeMounts := []corev1.VolumeMount{}
				for _, container := range pod.Spec.Containers {

					for _, VolumeMount := range container.VolumeMounts {
						// ??????????????????????????????????????????
						// ??????????????? / ??????
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

			log.Printf("?????????????????????%s; ?????????????????????Duplicate_mount_point: %+v\n", logfilepath_dir, Duplicate_mount_point)

			// ??????????????????????????????????????????0????????????????????????????????????????????????EmptyDir
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
				// ???????????????pod??????????????????????????????????????????
				for c_index, _ := range pod.Spec.Containers {
					pod.Spec.Containers[c_index].VolumeMounts = append(pod.Spec.Containers[c_index].VolumeMounts, VolumeMount)
				}
				// ???sidecar?????????????????????
				sidecarcontainer.VolumeMounts = append(sidecarcontainer.VolumeMounts, VolumeMount)
			} else {
				// ???sidecar?????????????????????
				sidecarcontainer.VolumeMounts = append(sidecarcontainer.VolumeMounts, Duplicate_mount_point...)
			}

		}

		// ?????????????????????
		programmenumber := configmap.Data["programmenumber"]
		if programmenumber == "2" {
			secret, _ := clientset.CoreV1().Secrets("logfile-operator-system").Get(context.TODO(), "elasticsearch-master-certs", metav1.GetOptions{})
			// fmt.Printf("secret: %+v\n", secret)

			// ????????????2??????filebeat????????????es8?????????https??????
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

			// ??????initcontainer??????filebeat???????????????
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
			// ???????????????????????????pod???
			pod.Spec.InitContainers = append(pod.Spec.InitContainers, eshttpscrtinitcontainer)
			// ???filebeat???????????????elasticsearch???https??????
			sidecarcontainer.VolumeMounts = append(sidecarcontainer.VolumeMounts, corev1.VolumeMount{
				Name:      "elasticsearch-master-certs",
				MountPath: "/usr/share/elasticsearch/config/certs",
			})
		}

		// ???????????????????????????pod???
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, sidecarinitcontainer)
		pod.Spec.Containers = append(pod.Spec.Containers, sidecarcontainer)
	}

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// PodSideCarMutate ?????? admission.DecoderInjector???
// ???????????????????????????
// ???????????????
func (v *PodSidecarMutate) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
