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

// 下面的注释用于生成 MutatingWebhookConfiguration 下webhook配置
//+kubebuilder:webhook:path=/mutate-huisebug-core-v1-pod,mutating=true,failurePolicy=fail,sideEffects=None,groups=core,resources=pods,verbs=create;update,versions=v1,name=mhuisebugpod.kb.io,admissionReviewVersions=v1
