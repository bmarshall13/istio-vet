/*
Copyright 2017 Aspen Mesh Authors.

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

// Package serviceassociation vets multiple service associations of pods in the
// mesh.
package serviceassociation

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	apiv1 "github.com/aspenmesh/istio-vet/api/v1"
	"github.com/aspenmesh/istio-vet/pkg/vetter/util"
)

const (
	vetterID                           = "serviceassociation"
	multipleServiceAssociationNoteType = "multiple-service-association"
	multipleServiceAssociationSummary  = "Multiple service association - ${service_list}"
	multipleServiceAssociationMsg      = "The services ${service_list} in namespace ${namespace}" +
		" are associated with the pod ${pod_name}. Consider updating the" +
		" service definitions ensuring the pod belongs to a single service."
)

// SvcAssociation implements Vetter interface
type SvcAssociation struct {
	info apiv1.Info
}

type endpointInfo struct {
	Namespace    string
	PodName      string
	ServiceNames []string
}

func createEndpointMap(e []corev1.Endpoints) map[string]endpointInfo {
	endpointMap := map[string]endpointInfo{}
	for _, ep := range e {
		for _, es := range ep.Subsets {
			for _, a := range es.Addresses {
				for _, p := range es.Ports {
					epMapKey := a.IP + ":" + fmt.Sprintf("%d", p.Port)
					if epInfo, ok := endpointMap[epMapKey]; !ok {
						endpointMap[epMapKey] = endpointInfo{
							Namespace:    ep.Namespace,
							PodName:      a.TargetRef.Name,
							ServiceNames: []string{ep.Name}}
					} else {
						svcs := append(epInfo.ServiceNames, ep.Name)
						epInfo.ServiceNames = svcs
						endpointMap[epMapKey] = epInfo
					}
				}
			}
		}
	}
	return endpointMap
}

// Vet returns the list of generated notes
func (m *SvcAssociation) Vet(c kubernetes.Interface) ([]*apiv1.Note, error) {
	notes := []*apiv1.Note{}
	endpoints, err := util.ListEndpointsInMesh(c)
	if err != nil {
		if n := util.IstioInitializerDisabledNote(err.Error(), vetterID,
			multipleServiceAssociationNoteType); n != nil {
			notes = append(notes, n)
			return notes, nil
		}
		return nil, err
	}

	epMap := createEndpointMap(endpoints)
	for _, v := range epMap {
		if len(v.ServiceNames) > 1 {
			notes = append(notes, &apiv1.Note{
				Type:    multipleServiceAssociationNoteType,
				Summary: multipleServiceAssociationSummary,
				Msg:     multipleServiceAssociationMsg,
				Level:   apiv1.NoteLevel_ERROR,
				Attr: map[string]string{
					"pod_name":     v.PodName,
					"namespace":    v.Namespace,
					"service_list": strings.Join(v.ServiceNames, ", ")}})
		}
	}

	for i := range notes {
		notes[i].Id = util.ComputeID(notes[i])
	}

	return notes, nil
}

// Info returns information about the vetter
func (m *SvcAssociation) Info() *apiv1.Info {
	return &m.info
}

// NewVetter returns "svcAssociation" which implements Vetter Interface
func NewVetter() *SvcAssociation {
	return &SvcAssociation{info: apiv1.Info{Id: vetterID, Version: "0.1.0"}}
}
