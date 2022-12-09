#!/bin/bash

genfile(){
kubectl get MutatingWebhookConfiguration logfile-operator-mutating-webhook-configuration -o yaml > MutatingWebhookConfiguration.yaml
}

#只替换mhuisebugpod.kb.io webhook相关配置
sedfile(){
sed -i "60,100 s/namespaceSelector: {}/namespaceSelector: \\n    matchLabels:\\n      pod-admission-webhook-injection: enabled/g" MutatingWebhookConfiguration.yaml
sed -i "60,100 s/scope: '\*'/scope: Namespaced/g" MutatingWebhookConfiguration.yaml
}
run(){
kubectl apply -f MutatingWebhookConfiguration.yaml
rm -rf MutatingWebhookConfiguration.yaml
}

genfile
sedfile
run

