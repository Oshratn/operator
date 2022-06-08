package mainhandler

import (
	"context"
	"fmt"
	"k8s-ca-websocket/cautils"
	"math/rand"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const CronjobTemplateName = "vulnscan-cronjob-template"

func (actionHandler *ActionHandler) setVulnScanCronJob() error {

	req := getVulnScanRequest(&actionHandler.command)

	name := fixK8sCronJobNameLimit(fmt.Sprintf("%s-%d", "vuln-scan-scheduled", rand.NewSource(time.Now().UnixNano()).Int63()))

	if err := createConfigMapForTriggerRequest(actionHandler.k8sAPI, name, req); err != nil {
		return err
	}

	jobTemplateObj, err := getCronJonTemplate(actionHandler.k8sAPI, CronjobTemplateName)
	if err != nil {
		return err
	}

	scanJobParams := getJobParams(&actionHandler.command)
	if scanJobParams == nil || scanJobParams.CronTabSchedule == "" {
		return fmt.Errorf("setVulnScanCronJob: CronTabSchedule not found")
	}
	setCronJobForTriggerRequest(jobTemplateObj, name, scanJobParams.CronTabSchedule, actionHandler.command.JobTracking.JobID)

	if _, err := actionHandler.k8sAPI.KubernetesClient.BatchV1().CronJobs(cautils.CA_NAMESPACE).Create(context.Background(), jobTemplateObj, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (actionHandler *ActionHandler) updateVulnScanCronJob() error {
	scanJobParams := getJobParams(&actionHandler.command)
	if scanJobParams == nil || scanJobParams.CronTabSchedule == "" {
		return fmt.Errorf("updateVulnScanCronJob: CronTabSchedule not found")
	}
	if scanJobParams.JobName == "" {
		return fmt.Errorf("updateVulnScanCronJob: jobName not found")
	}

	jobTemplateObj, err := actionHandler.k8sAPI.KubernetesClient.BatchV1().CronJobs(cautils.CA_NAMESPACE).Get(context.Background(), scanJobParams.JobName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	jobTemplateObj.Spec.Schedule = scanJobParams.CronTabSchedule
	if jobTemplateObj.Spec.JobTemplate.Spec.Template.Annotations == nil {
		jobTemplateObj.Spec.JobTemplate.Spec.Template.Annotations = make(map[string]string)
	}
	jobTemplateObj.Spec.JobTemplate.Spec.Template.Annotations[armoJobIDAnnotation] = actionHandler.command.JobTracking.JobID

	_, err = actionHandler.k8sAPI.KubernetesClient.BatchV1().CronJobs(cautils.CA_NAMESPACE).Update(context.Background(), jobTemplateObj, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (actionHandler *ActionHandler) deleteVulnScanCronJob() error {

	scanJobParams := getJobParams(&actionHandler.command)
	if scanJobParams == nil || scanJobParams.JobName == "" {
		return fmt.Errorf("deleteVulnScanCronJob: CronTabSchedule not found")
	}

	return actionHandler.deleteCronjob(scanJobParams.JobName)

}

func (actionHandler *ActionHandler) deleteCronjob(name string) error {
	if err := actionHandler.k8sAPI.KubernetesClient.BatchV1().CronJobs(cautils.CA_NAMESPACE).Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	if err := actionHandler.k8sAPI.KubernetesClient.CoreV1().ConfigMaps(cautils.CA_NAMESPACE).Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil

}
