package main

import(
    "context"
	"flag"
	"fmt"
	"os"
    "strings"
    "time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

    corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    appsv1 "k8s.io/api/apps/v1"
)

func main() {

    var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", home+"/.kube/config", "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Printf("Error building kubeconfig: %s\n", err.Error())
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Error building kubernetes clientset: %s\n", err.Error())
		os.Exit(1)
	}

	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error listing pods: %s\n", err.Error())
		os.Exit(1)
	}

    for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "database") {
            fmt.Printf("database pod: %s found\n", pod.Name)

            deployment := getDeploymentName(&pod, clientset)
            fmt.Printf("deployment name: %s\n", deployment)

            err := updateDeployment(clientset, pod.Namespace, deployment)
            if err != nil {
                fmt.Printf("deployment update error: %s\n", err)
            }
		}
    }
}

func getDeploymentName(pod *corev1.Pod, clientset *kubernetes.Clientset) string {
    for _, ownerRef := range pod.OwnerReferences {
        if ownerRef.Kind == "Deployment" {
            fmt.Printf("Deployment Name: %s\n", ownerRef.Name)
            return ownerRef.Name
        } else if ownerRef.Kind == "ReplicaSet" {
            replicaSet, err := clientset.AppsV1().ReplicaSets(pod.Namespace).Get(context.TODO(), ownerRef.Name, metav1.GetOptions{})
            if err != nil {
                fmt.Printf("Error getting ReplicaSet: %s", err.Error())
            }
            deploymentName := getDeploymentNameFromReplicaSet(replicaSet)

            return deploymentName
        }
    }
    return "N/A"
}

func getDeploymentNameFromReplicaSet(rs *appsv1.ReplicaSet) string {
    for _, ownerRef := range rs.OwnerReferences {
        if ownerRef.Kind == "Deployment" {
            return ownerRef.Name
        }
    }
    return "N/A"
}

func updateDeployment(clientset *kubernetes.Clientset, namespace string, deploymentName string) error {
    deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
    if err != nil {
        return fmt.Errorf("failed to get Deployment: %v", err)
    }

    if deployment.Spec.Template.Annotations == nil {
        deployment.Spec.Template.Annotations = make(map[string]string)
    }
    deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

    _, err = clientset.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
    if err != nil {
        return fmt.Errorf("failed to update Deployment: %v", err)
    }

    return nil
}
