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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LogFileSpec defines the desired state of LogFile
type LogFileSpec struct {
	// 方案序号
	ProgrammeNum int `json:"programmenum,"`

	// 密码认证
	ELASTIC_PASSWORD string `json:"elastic_password,"`
	KIBANA_PASSWORD  string `json:"kibana_password,"`
	// 服务持久化使用的storageclass
	StorageClassName string `json:"storageClassName,"`
	// 申请空间大小
	ResourceStorage *ResourceStorage `json:"resourcestorage,omitempty"`
	// 暴露主机端口服务
	NodePortS *NodePortS `json:"nodePorts,omitempty"`
}

// LogFileStatus defines the observed state of LogFile
type LogFileStatus struct {
	Status string `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LogFile is the Schema for the logfiles API
type LogFile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LogFileSpec   `json:"spec,omitempty"`
	Status LogFileStatus `json:"status,omitempty"`
}

type ResourceStorage struct {
	Elasticsearch string `json:"elasticsearch"`
	Kafka         string `json:"kafka"`
	Zookeeper     string `json:"zookeeper"`
}

type NodePortS struct {
	Elasticsearch int `json:"elasticsearch"`
	Kibana        int `json:"kibana"`
}

//+kubebuilder:object:root=true

// LogFileList contains a list of LogFile
type LogFileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LogFile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LogFile{}, &LogFileList{})
}
