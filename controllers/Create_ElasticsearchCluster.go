package controllers

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"

	apiv1 "github.com/huisebug/logfile-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *LogFileReconciler) ElasticsearchClusterCretePodDisruptionBudget(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {

	customizelog := logger.WithValues("func", "ElasticsearchClusterCretePodDisruptionBudget")
	meta.Name += "-pdb"
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: meta,
		Spec: policyv1.PodDisruptionBudgetSpec{
			MaxUnavailable: &intstr.IntOrString{
				Type:   0,
				IntVal: 1,
				StrVal: "",
			},
			Selector: metav1.SetAsLabelSelector(labels),
		},
	}

	// 级联删除
	customizelog.Info("set pdb reference")
	if err := controllerutil.SetControllerReference(logfile, pdb, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	// 新建
	if err := r.Create(ctx, pdb); err != nil {
		return err
	}

	customizelog.Info("create pdb success", "name", typesname.String())

	return nil
}

func (r *LogFileReconciler) ElasticsearchClusterCreteSecret(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {

	customizelog := logger.WithValues("func", "ElasticsearchClusterCreteSecret")
	meta.Name += "-certs"

	tlscrtbase64 := "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURxRENDQXBDZ0F3SUJBZ0lSQUxZazFXa0lHMTNxVEgyQXN3enlQbEl3RFFZSktvWklodmNOQVFFTEJRQXcKR3pFWk1CY0dBMVVFQXhNUVpXeGhjM1JwWTNObFlYSmphQzFqWVRBZUZ3MHlNakV4TVRFd016RXhORFJhRncweQpNekV4TVRFd016RXhORFJhTUI4eEhUQWJCZ05WQkFNVEZHVnNZWE4wYVdOelpXRnlZMmd0YldGemRHVnlNSUlCCklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUFyUDVPM0dEVlU3Q0FiRTRJcVkwUVFnbTgKdG5obTB5NkRiM2dsSnZ3cEtyRkljV1Nxd0Q1UEsyaWpBMDJaNGt0TXg0RGFmZ2p4dGJUaUdONEVGTVhDUUxHWQp6THBlV1BSMHVJOGIxSXk3MG1jZE5YZTJrdnk2SzExalV0TGNpUXNSMkV5aEY2c3YwR1dWNC9JdDh1OVIvOHN1CmxPM3dOc2dCUHNZZFBvMFNsalVSM3crdEdPL1VRb3NUeFFzTXpVRTZFYXNiZG05NUpJVk5JckU1dSt0SkZkUzAKaGtQcmNwRGRzWlB3aWtpZi9LM0JvM3F2K2dFdWMxcXI4T055eE1KMmhOcjUybkxlM21PaTVHb1ZUd1hMd0ZESwo1RUhQNWNobml2ZjQ5eFY1N3NWcmd4TVk5Zml4RWJIVlA0QjZqbEtlRzIwVXcyYUdsak5rUERnOHQ1NXNqUUlECkFRQUJvNEhpTUlIZk1BNEdBMVVkRHdFQi93UUVBd0lGb0RBZEJnTlZIU1VFRmpBVUJnZ3JCZ0VGQlFjREFRWUkKS3dZQkJRVUhBd0l3REFZRFZSMFRBUUgvQkFJd0FEQWZCZ05WSFNNRUdEQVdnQlFQNGp1Qzk4UVZDU2VGdm5mdApNMTFsMXdIVFlqQi9CZ05WSFJFRWVEQjJnaFJsYkdGemRHbGpjMlZoY21Ob0xXMWhjM1JsY29Jc1pXeGhjM1JwClkzTmxZWEpqYUMxdFlYTjBaWEl1Ykc5blptbHNaUzF2Y0dWeVlYUnZjaTF6ZVhOMFpXMkNNR1ZzWVhOMGFXTnoKWldGeVkyZ3RiV0Z6ZEdWeUxteHZaMlpwYkdVdGIzQmxjbUYwYjNJdGMzbHpkR1Z0TG5OMll6QU5CZ2txaGtpRwo5dzBCQVFzRkFBT0NBUUVBWTJFZk5jZmQ3eUFuS2JuRFZVZFJoL2FIZ0VvOTZQdklVZ0lhTjJmQnE2QSt5MWJFCkFYUVZYODhHZ1RJK25XNnE0T28xOXYzRmVFNkJKQzVWQk1iSStJZ2dwdW5pVnQxdVFFM0ZVRmhBUHVZOTVjWTAKN254OFNCNlcvTkpvWXgzb2JLWjBxWjVaM0hkQ1R6TjFmeXpzMWtkSk43eEtRNE5QbnVGby9JUDVvVGNNTlpaTQphSFVTZWd5US83QWM1UUd0R1h6YytlTG4veUlUMGZCdGNlcVpSRWhaMnEybWNySjlwQTE1QjQ1Nmx3WmNiR0k2CmhKa2ZtRXY4MS92R2lsQnd0Yk15QlB6T0dnbHBaVmVUd1FTMGczYnozV1FsSlR3elB0bitRSVdvY2YwZVRFWE4Ka0FzRFFtdUNKa1MzWTQrNStkUVB1ODRyS3RNTC9yaXBnM2Uwanc9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
	tlskeybase64 := "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBclA1TzNHRFZVN0NBYkU0SXFZMFFRZ204dG5obTB5NkRiM2dsSnZ3cEtyRkljV1NxCndENVBLMmlqQTAyWjRrdE14NERhZmdqeHRiVGlHTjRFRk1YQ1FMR1l6THBlV1BSMHVJOGIxSXk3MG1jZE5YZTIKa3Z5NksxMWpVdExjaVFzUjJFeWhGNnN2MEdXVjQvSXQ4dTlSLzhzdWxPM3dOc2dCUHNZZFBvMFNsalVSM3crdApHTy9VUW9zVHhRc016VUU2RWFzYmRtOTVKSVZOSXJFNXUrdEpGZFMwaGtQcmNwRGRzWlB3aWtpZi9LM0JvM3F2CitnRXVjMXFyOE9OeXhNSjJoTnI1Mm5MZTNtT2k1R29WVHdYTHdGREs1RUhQNWNobml2ZjQ5eFY1N3NWcmd4TVkKOWZpeEViSFZQNEI2amxLZUcyMFV3MmFHbGpOa1BEZzh0NTVzalFJREFRQUJBb0lCQVFDUmY3OHlTZHVDN1RQRwpaQWVURzRzdUQrU0NFRmhqakg2TnRaNkI0SnA3UnVxb1BNUUV0eU45WGgrbE9wS1FLMGNqa1RPeng3QU1aVnU1CkVKcWNJZ3lVdndyR1BvWDJDWDFXY1Q4MkVUd3o1ZmhDTFRNSkE3bE5tZGxkSXU3TDhOeU1jVDhZbWltMy9Ja0sKVkhuakZ2aC96Uk9idlZoSnF3U3BHSlltTXg4TDU2bDMxUFdPK3k2VmY3OVZuaDdkWGFnSUlsNDhJYWtGenRYTApqaGd1cGFMSmhrWDQrQmswYXdLTUVZS2RHSW5sTEJiZFVEU1dYUW83N1pWb0YwbFNNVHJJZlU0TDNJRzhpWXQrCmtuZ0k3RzlSRzFleHNoYzNmSm53aFlMd0pQWWZibndiZHJVZ1pIa0cvaC9ONVJ0NmxXRjA0Q3p0K3BxQUNvZ1AKNjZCVjV6V2RBb0dCQU9ZNEtoa0s0MmZJTGhmWlVqOEc5SG10Q0ZWa3E4RWZoRDVoNU9jclU3NVp2MXJjcHEwUAp2TERDZWNLYitVUWoyVEUyQnpQcHZBMjlkdjVERzI4a0lnamZlMk1rbmtBa3Z6RVlWSVlBRU5yQURMOXRKMlRMCmNEVmcyclpDWm5QUGp1L0lHVFI4T2d1dmtIMkwxS1pZMWh4TDhzNnZPS0RWUnExV2VZVUxSZktQQW9HQkFNQmQKbktrYm50YVVRaW54d3hXSUtBSW5NdVVRTll2QzltTUZ5M2JyKytBLy9jY3BKdHZsWW0rajNsZm5UUTRqc214dApiOVdySjh2TE0vb3c1WkdhTWtVRXltdHhaTkI0YVZZRTJPemZDUjJCVDE1YlpwbUZNVmxNQ3kyVndmQmpxSG5SCk1ObklxdW83dXZEVWM0ZFdZeTV0L1lFNVAzSTM0NVFkTFM1Kzh3MGpBb0dBQVZ3Rmk1NVAxM1lNSjZIbDVXOWkKRkRIY1lieTFjdTkvdFdxWWtuRGtEclN5OTVOai9KT2lOcHovWVJIUXVBRktNQXMwb2E3WXFIQWMrc1ZrclJSVwppeHpldXFnbHN4VkVkOExBQlFhTkV1MmRaYWY4V3BFRStadTN6dW0zZHltYm0zamdCVHBTa1cwWStsVFFEYWRxCnBFSWlqZXZrOXJZcnM2eFdEVjRTcktzQ2dZQXAxVEc0Wk5WSi9Mdld1MGlkYWhxcFBUVUlNMW94cHBoR09JQmkKd0RicU1ZQlN5MVEwQmRJK1RQaVJUUytvbjRLeHFhcmtZSEFyRldtY1F2M3BpQXJlajRnbGpXZExIcVJwbkd4QQpOdENZcGdKSWxyL2RLdVhzY1drTTVNQmtNb2YwMWRVMXh6bkQ3bkZjNWhhcG05Tzl5UldVQUlzWG42ZlNFZlk5CllrWWcyUUtCZ0FwMUIvRTFlTzZjSHg2UlUxVHdUK05oZkJidzNxempkUkpJamhXUVQvTHVRYUJYUWFPZUtjRmoKa1pvM1d6QW13Z3JIN3c4MWwwNlVzQnU4eGViQU9BNzgwSHk3TjB2NVpCbkluT1AzMzR6c0F1cExkZWMzTVBsNwpzczFnak95engrdTdIK0d1Mi9rRjZXb2lzUVFuVzlkSkhvZDFSSDhqV0dTL2IwSW9kUjIxCi0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg=="
	cacrtbase64 := "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURJakNDQWdxZ0F3SUJBZ0lSQUpsbGxJSEl2ZHorNWlGajk4TnJCdU13RFFZSktvWklodmNOQVFFTEJRQXcKR3pFWk1CY0dBMVVFQXhNUVpXeGhjM1JwWTNObFlYSmphQzFqWVRBZUZ3MHlNakV4TVRFd016RXhORFJhRncweQpNekV4TVRFd016RXhORFJhTUJzeEdUQVhCZ05WQkFNVEVHVnNZWE4wYVdOelpXRnlZMmd0WTJFd2dnRWlNQTBHCkNTcUdTSWIzRFFFQkFRVUFBNElCRHdBd2dnRUtBb0lCQVFDbENieDFGWHRWa05vZDFRRG05TGluZUw1UXlQY08KN1JraGVoTm53TU9SNUU2Q2Uva1dlTkVyaTAyNXQzK0lrME1lcXJvT3ZNanljT3FJekZEVzJXYjlaQ1FaM1JpTQpPQWlDbTNpd0lWa0M4ZTRDRkNpOGV6N2JPbUR1TVFsY2xWWitPTWgzT0V1UUVLRW9aVGI2bjBobXhOT0FqVGVHCnNNOXN4ZldLYXZRVWFjTjFRQzA0bGtkM1B3OU1haFhSTDVyQUdnU2MrcElleUEzQ3NLdVlSeUhzSlh3Um01M2gKR0ZQRXZDM3V5S3JVczRxQk43WUJDUlQ1NUhJNmdwcnU3WUhUcDM0WFhiSmJPWEFhcFAvV1JqRHJkeFRoaS93YQphTEptNHZJS0R4NWZCb1FDdTBGMzhybzV6eThBOU1oenZwNXpCbXRBNlVhZTdUMGFIaklHeW1OWEFnTUJBQUdqCllUQmZNQTRHQTFVZER3RUIvd1FFQXdJQ3BEQWRCZ05WSFNVRUZqQVVCZ2dyQmdFRkJRY0RBUVlJS3dZQkJRVUgKQXdJd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVUQrSTdndmZFRlFrbmhiNTM3VE5kWmRjQgowMkl3RFFZSktvWklodmNOQVFFTEJRQURnZ0VCQUdsVmgxOVFlR2hVVHhtZGx1UTRUK21aanc0OTJraEU1T25KCm5mWG1vSU5pTlFmZkVxUEtleHB4RGRrME4yWVc1K2JESUxmYysrQzU5TDFQS1dZTTVxd2lEeEFrN3FLMExVSksKNWtLUGdFNzFDYm1ZSCt3ZXpobmNBdkpyd3Q2ME5RTjlMTCt3d1F5SUdQVmZXS1ZOWDVueXBDNlliL0pOb1dyZwo1N01iQk9iRDdGVGg0MWY5NG1GWWxEL3U0L1hGTUF4czZLekduU3JSRmNqZ1prUnV2RnpqM3AvZytmbW1xQzZ1CjJQS0JlOWRKbkVGcW4vVTlYN3dDaWRoandCQmFuZUdNTmZIYjFyaHduNm1FSDdKTTN3WEVCeW5UZjRja3hLQ2kKNHNjdnRESlZSZXNZQWRDcVBsWUF5QW5tQmIyNFZEQ1Z6czZjWjloNzRmdk5INzhPc3RNPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="

	tlscrt, _ := base64.RawStdEncoding.DecodeString(tlscrtbase64)
	tlskey, _ := base64.RawStdEncoding.DecodeString(tlskeybase64)
	cacrt, _ := base64.RawStdEncoding.DecodeString(cacrtbase64)
	// fmt.Println(string(tlscrt))
	// fmt.Println(string(tlskey))
	// fmt.Println(string(cacrt))

	tmpmap := make(map[string][]byte)
	tmpmap["tls.crt"] = tlscrt
	tmpmap["tls.key"] = tlskey
	tmpmap["ca.crt"] = cacrt

	secret := &corev1.Secret{
		ObjectMeta: meta,
		Data:       tmpmap,
		Type:       corev1.SecretTypeTLS,
	}

	// 级联删除
	customizelog.Info("set secret reference")
	if err := controllerutil.SetControllerReference(logfile, secret, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	// 新建
	if err := r.Create(ctx, secret); err != nil {
		return err
	}

	customizelog.Info("create secret success", "name", typesname.String())

	return nil
}

func (r *LogFileReconciler) ElasticsearchClusterCreteService(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {

	customizelog := logger.WithValues("func", "ElasticsearchClusterCreteService")
	// 暴露NodePort
	log.Printf("%+v\n", logfile.Spec.NodePortS)
	headlessmeta := meta.DeepCopy()
	headlessmeta.Name = headlessmeta.Name + "-headless"

	serviceheadless := &corev1.Service{
		ObjectMeta: *headlessmeta,
		Spec: corev1.ServiceSpec{
			ClusterIP:                "None",
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     int32(9200),
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name:     "transport",
					Port:     int32(9300),
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: labels,
		},
	}

	service := &corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Type:                     corev1.ServiceTypeNodePort,
			Selector:                 labels,
			PublishNotReadyAddresses: false,
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     int32(9200),
					Protocol: corev1.ProtocolTCP,
					NodePort: int32(logfile.Spec.NodePortS.Elasticsearch),
				},
				{
					Name:     "transport",
					Port:     int32(9300),
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	// 级联删除service
	customizelog.Info("set serviceheadless reference")
	if err := controllerutil.SetControllerReference(logfile, serviceheadless, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	if err := controllerutil.SetControllerReference(logfile, service, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}
	// 新建service
	if err := r.Create(ctx, serviceheadless); err != nil {
		return err
	}
	if err := r.Create(ctx, service); err != nil {
		return err
	}

	customizelog.Info("create serviceheadless and service success", "name", typesname.String())

	return nil
}

func (r *LogFileReconciler) ElasticsearchClusterCreteStatefulSet(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {

	var AutomountServiceAccountToken = true
	var EnableServiceLinks = true

	var InitcontainerPrivileged = true
	var RunAsNonRoot = true

	customizelog := logger.WithValues("func", "ElasticsearchCreteStatefulSet")

	Command := []byte(
		`
set -e

# Exit if ELASTIC_PASSWORD in unset
if [ -z "${ELASTIC_PASSWORD}" ]; then
echo "ELASTIC_PASSWORD variable is missing, exiting"
exit 1
fi

# If the node is starting up wait for the cluster to be ready (request params: "wait_for_status=green&timeout=1s" )
# Once it has started only check that the node itself is responding
START_FILE=/tmp/.es_start_file

# Disable nss cache to avoid filling dentry cache when calling curl
# This is required with Elasticsearch Docker using nss < 3.52
export NSS_SDB_USE_CACHE=no

http () {
local path="${1}"
local args="${2}"
set -- -XGET -s

if [ "$args" != "" ]; then
	set -- "$@" $args
fi

set -- "$@" -u "elastic:${ELASTIC_PASSWORD}"

curl --output /dev/null -k "$@" "https://127.0.0.1:9200${path}"
}

if [ -f "${START_FILE}" ]; then
echo 'Elasticsearch is already running, lets check the node is healthy'
HTTP_CODE=$(http "/" "-w %{http_code}")
RC=$?
if [[ ${RC} -ne 0 ]]; then
	echo "curl --output /dev/null -k -XGET -s -w '%{http_code}' \${BASIC_AUTH} https://127.0.0.1:9200/ failed with RC ${RC}"
	exit ${RC}
fi
# ready if HTTP code 200, 503 is tolerable if ES version is 6.x
if [[ ${HTTP_CODE} == "200" ]]; then
	exit 0
elif [[ ${HTTP_CODE} == "503" && "8" == "6" ]]; then
	exit 0
else
	echo "curl --output /dev/null -k -XGET -s -w '%{http_code}' \${BASIC_AUTH} https://127.0.0.1:9200/ failed with HTTP code ${HTTP_CODE}"
	exit 1
fi

else
echo 'Waiting for elasticsearch cluster to become ready (request params: "wait_for_status=green&timeout=1s" )'
if http "/_cluster/health?wait_for_status=green&timeout=1s" "--fail" ; then
	touch ${START_FILE}
	exit 0
else
	echo 'Cluster is not yet ready (request params: "wait_for_status=green&timeout=1s" )'
	exit 1
fi
fi
	`)

	// 申请storageclass的大小
	log.Printf("%+v\n", logfile.Spec.ResourceStorage)
	var resourceStorage, _ = resource.ParseQuantity(logfile.Spec.ResourceStorage.Elasticsearch)

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
			PodManagementPolicy: appsv1.PodManagementPolicyType("Parallel"),
			Replicas:            pointer.Int32Ptr(3),
			Selector:            metav1.SetAsLabelSelector(labels),
			ServiceName:         meta.Name + "-headless",
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup:   pointer.Int64(1000),
						RunAsUser: pointer.Int64(1000),
					},
					AutomountServiceAccountToken: &AutomountServiceAccountToken,
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
					TerminationGracePeriodSeconds: pointer.Int64(120),
					Volumes: []corev1.Volume{
						{
							Name: meta.Name + "-certs",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: meta.Name + "-certs",
								},
							},
						},
					},
					EnableServiceLinks: &EnableServiceLinks,
					InitContainers: []corev1.Container{
						{
							Name: "configure-sysctl",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:  pointer.Int64(0),
								Privileged: &InitcontainerPrivileged,
							},
							Image:           "registry.cn-hangzhou.aliyuncs.com/huisebug/logfile-operator:elasticsearch-8.5.0",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"sysctl", "-w", "vm.max_map_count=262144"},
						},
					},

					Containers: []corev1.Container{
						{
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:    pointer.Int64(1000),
								RunAsNonRoot: &RunAsNonRoot,
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										"All",
									},
								},
							},
							Name:            "elasticsearch",
							Image:           "registry.cn-hangzhou.aliyuncs.com/huisebug/logfile-operator:elasticsearch-8.5.0",
							ImagePullPolicy: corev1.PullIfNotPresent,
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"bash",
											"-c",
											string(Command),
										},
									},
								},
								FailureThreshold:    *pointer.Int32(3),
								InitialDelaySeconds: *pointer.Int32(10),
								PeriodSeconds:       *pointer.Int32(10),
								SuccessThreshold:    *pointer.Int32(3),
								TimeoutSeconds:      *pointer.Int32(5),
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.Protocol("TCP"),
									ContainerPort: int32(9200),
								},
								{
									Name:          "transport",
									Protocol:      corev1.Protocol("TCP"),
									ContainerPort: int32(9300),
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("1000m"),
									"memory": resource.MustParse("2Gi"),
								},
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("1000m"),
									"memory": resource.MustParse("2Gi"),
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "TZ",
									Value: "Asia/Shanghai",
								},
								{
									Name:  "ELASTIC_PASSWORD",
									Value: logfile.Spec.ELASTIC_PASSWORD,
								},
								{
									Name:  "KIBANA_PASSWORD",
									Value: logfile.Spec.KIBANA_PASSWORD,
								},
								{
									Name: "node.name",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name:  "cluster.initial_master_nodes",
									Value: fmt.Sprintf("%s-0,%s-1,%s-2,", meta.Name, meta.Name, meta.Name),
								},
								{
									Name:  "node.roles",
									Value: "master,data,ingest,",
								},
								{
									Name:  "discovery.seed_hosts",
									Value: meta.Name + "-headless",
								},
								{
									Name:  "cluster.name",
									Value: "elasticsearch",
								},
								{
									Name:  "network.host",
									Value: "0.0.0.0",
								},
								{
									Name:  "xpack.security.enabled",
									Value: "true",
								},
								{
									Name:  "xpack.security.transport.ssl.enabled",
									Value: "true",
								},
								{
									Name:  "xpack.security.http.ssl.enabled",
									Value: "true",
								},
								{
									Name:  "xpack.security.transport.ssl.verification_mode",
									Value: "certificate",
								},
								{
									Name:  "xpack.security.transport.ssl.key",
									Value: "/usr/share/elasticsearch/config/certs/tls.key",
								},
								{
									Name:  "xpack.security.transport.ssl.certificate",
									Value: "/usr/share/elasticsearch/config/certs/tls.crt",
								},
								{
									Name:  "xpack.security.transport.ssl.certificate_authorities",
									Value: "/usr/share/elasticsearch/config/certs/ca.crt",
								},
								{
									Name:  "xpack.security.http.ssl.key",
									Value: "/usr/share/elasticsearch/config/certs/tls.key",
								},
								{
									Name:  "xpack.security.http.ssl.certificate",
									Value: "/usr/share/elasticsearch/config/certs/tls.crt",
								},
								{
									Name:  "xpack.security.http.ssl.certificate_authorities",
									Value: "/usr/share/elasticsearch/config/certs/ca.crt",
								},
								{
									Name:  "xpack.license.self_generated.type",
									Value: "basic",
								},
							},

							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      meta.Name,
									MountPath: "/usr/share/elasticsearch/data",
								},
								{
									Name:      meta.Name + "-certs",
									MountPath: "/usr/share/elasticsearch/config/certs",
									ReadOnly:  true,
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
func (r *LogFileReconciler) ElasticsearchClusterKibanaUserCreteJob(ctx context.Context, logfile *apiv1.LogFile, typesname types.NamespacedName, meta metav1.ObjectMeta, labels map[string]string) error {

	customizelog := logger.WithValues("func", "ElasticsearchClusterKibanaUserCreteJob")
	job := &batchv1.Job{
		ObjectMeta: meta,
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "elasticsearch-master-certs",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "elasticsearch-master-certs",
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "set-kibana-user-password",
							Image:           "registry.cn-hangzhou.aliyuncs.com/huisebug/logfile-operator:elasticsearch-8.5.0",
							ImagePullPolicy: "IfNotPresent",
							Env: []corev1.EnvVar{
								{
									Name:  "ELASTIC_PASSWORD",
									Value: logfile.Spec.ELASTIC_PASSWORD,
								},
								{
									Name:  "KIBANA_PASSWORD",
									Value: logfile.Spec.KIBANA_PASSWORD,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "elasticsearch-master-certs",
									MountPath: "/usr/share/elasticsearch/config/certs",
									ReadOnly:  true,
								},
							},
							Command: []string{
								"/bin/bash",
								"-c",
							},
							Args: []string{
								`until curl -s -X POST --cacert /usr/share/elasticsearch/config/certs/ca.crt -u "elastic:${ELASTIC_PASSWORD}" -H "Content-Type: application/json" https://elasticsearch-master:9200/_security/user/kibana_system/_password -d "{\"password\":\"${KIBANA_PASSWORD}\"}" | grep -q "^{}"; do sleep 10; done;`,
							},
						},
					},
				},
			},
		},
	}

	// 级联删除job
	customizelog.Info("set job reference")
	if err := controllerutil.SetControllerReference(logfile, job, r.Scheme); err != nil {
		customizelog.Error(err, "SetControllerReference error")
		return err
	}

	// 新建job
	if err := r.Create(ctx, job); err != nil {
		return err
	}

	customizelog.Info("create job success", "name", typesname.String())

	return nil
}
