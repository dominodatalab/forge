package buildjob

import (
	"strings"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1alpha1 "github.com/dominodatalab/forge/api/v1alpha1"
)

type StatusUpdate struct {
	Name          string            `json:"name"`
	Annotations   map[string]string `json:"annotations"`
	ObjectLink    string            `json:"objectLink"`
	PreviousState string            `json:"previousState"`
	CurrentState  string            `json:"currentState"`
	ErrorMessage  string            `json:"errorMessage"`
	ImageURLs     []string          `json:"imageURLs"`
}

func (j *Job) transitionToBuilding(cib *apiv1alpha1.ContainerImageBuild) (*apiv1alpha1.ContainerImageBuild, error) {
	cib.Status.SetState(apiv1alpha1.BuildStateBuilding)
	cib.Status.BuildStartedAt = &metav1.Time{Time: time.Now()}

	return j.updateStatus(cib)
}

func (j *Job) transitionToComplete(cib *apiv1alpha1.ContainerImageBuild, images []string) error {
	cib.Status.SetState(apiv1alpha1.BuildStateCompleted)
	cib.Status.ImageURLs = images
	cib.Status.BuildCompletedAt = &metav1.Time{Time: time.Now()}

	_, err := j.updateStatus(cib)
	return err
}

func (j *Job) transitionToFailure(cib *apiv1alpha1.ContainerImageBuild, err error) error {
	cib.Status.SetState(apiv1alpha1.BuildStateFailed)
	cib.Status.ErrorMessage = err.Error()
	cib.Status.BuildCompletedAt = &metav1.Time{Time: time.Now()}

	_, err = j.updateStatus(cib)
	return err
}

func (j *Job) updateStatus(cib *apiv1alpha1.ContainerImageBuild) (*apiv1alpha1.ContainerImageBuild, error) {
	cib, err := j.clientforge.ContainerImageBuilds(j.namespace).UpdateStatus(cib)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update status")
	}

	if j.producer != nil {
		update := &StatusUpdate{
			Name:          cib.Name,
			Annotations:   cib.Annotations,
			ObjectLink:    strings.TrimSuffix(cib.GetSelfLink(), "/status"),
			PreviousState: string(cib.Status.PreviousState),
			CurrentState:  string(cib.Status.State),
			ImageURLs:     cib.Status.ImageURLs,
			ErrorMessage:  cib.Status.ErrorMessage,
		}
		if err := j.producer.Publish(update); err != nil {
			return nil, errors.Wrap(err, "unable to publish message")
		}
	}

	return cib, nil
}
