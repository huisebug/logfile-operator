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
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var logfilelog = logf.Log.WithName("logfile-resource")

func (r *LogFile) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-api-huisebug-org-v1-logfile,mutating=true,failurePolicy=fail,sideEffects=None,groups=api.huisebug.org,resources=logfiles,verbs=create;update,versions=v1,name=mlogfile.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &LogFile{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *LogFile) Default() {
	logfilelog.Info("default", "name", r.Name)
	Elasticsearchstorage := "100Gi"
	Kafkastorage := "10Gi"
	Zookeeperstorage := "10Gi"
	// 申请storageclass的大小

	if r.Spec.ResourceStorage == nil {
		DefaultRS := ResourceStorage{}
		DefaultRS.Elasticsearch = Elasticsearchstorage
		DefaultRS.Kafka = Kafkastorage
		DefaultRS.Zookeeper = Zookeeperstorage
		r.Spec.ResourceStorage = &DefaultRS
	}

	// 对未传递的申请空间大小进行赋值
	v := reflect.ValueOf(*r.Spec.ResourceStorage)
	t := reflect.TypeOf(*r.Spec.ResourceStorage)
	count := v.NumField()
	for i := 0; i < count; i++ {
		f := v.Field(i) //字段值
		if f.String() == "" {
			switch t.Field(i).Name {
			case "Elasticsearch":
				logfilelog.Info("default", "name", r.Name)
				r.Spec.ResourceStorage.Elasticsearch = Elasticsearchstorage
			case "Kafka":
				r.Spec.ResourceStorage.Kafka = Kafkastorage
			case "Zookeeper":
				r.Spec.ResourceStorage.Zookeeper = Zookeeperstorage
			}
		}
	}
	ElasticsearchNodePort := 31920
	KibanaNodePort := 31056
	if r.Spec.NodePortS == nil {
		Defaultports := NodePortS{}
		Defaultports.Elasticsearch = ElasticsearchNodePort
		Defaultports.Kibana = KibanaNodePort
		r.Spec.NodePortS = &Defaultports
	}

	// 对未传递的nodePort进行赋值
	nodeportsv := reflect.ValueOf(*r.Spec.NodePortS)
	nodeportst := reflect.TypeOf(*r.Spec.NodePortS)
	nodeportscount := nodeportsv.NumField()
	for i := 0; i < nodeportscount; i++ {
		f := nodeportsv.Field(i) //字段值
		if f.Int() == 0 {
			switch nodeportst.Field(i).Name {
			case "Elasticsearch":
				r.Spec.NodePortS.Elasticsearch = ElasticsearchNodePort
			case "Kibana":
				r.Spec.NodePortS.Kibana = KibanaNodePort

			}
		}
	}

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-api-huisebug-org-v1-logfile,mutating=false,failurePolicy=fail,sideEffects=None,groups=api.huisebug.org,resources=logfiles,verbs=create;update,versions=v1,name=vlogfile.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &LogFile{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *LogFile) ValidateCreate() error {
	logfilelog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return r.validate()

}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *LogFile) ValidateUpdate(old runtime.Object) error {
	logfilelog.Info("validate update", "name", r.Name)

	var allErrs field.ErrorList
	err := field.Invalid(field.NewPath("LogFile").Child("Spec"),
		r.Name,
		"不允许进行更新操作，请执行delete后在执行create")
	allErrs = append(allErrs, err)
	return apierrors.NewInvalid(
		schema.GroupKind{Group: "apiv1.huisebug.org", Kind: "LogFile"},
		r.Name,
		allErrs)

}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *LogFile) ValidateDelete() error {
	logfilelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *LogFile) validate() error {
	var allErrs field.ErrorList
	if r.Spec.ProgrammeNum > 6 || r.Spec.ProgrammeNum <= 0 {
		err := field.Invalid(field.NewPath("Spec").Child("ProgrammeNum"),
			r.Spec.ProgrammeNum,
			"当前仅支持6种部署方案")
		allErrs = append(allErrs, err)
		return apierrors.NewInvalid(
			schema.GroupKind{Group: "apiv1.huisebug.org", Kind: "LogFile"},
			r.Name,
			allErrs)
	}

	return nil
}
