package upcloud

import (
	"fmt"
	"github.com/UpCloudLtd/upcloud-go-sdk/upcloud/request"
	"github.com/UpCloudLtd/upcloud-go-sdk/upcloud/service"
	"log"
	"strings"
)

// Artifact represents a template of a storage as the result of a Packer build
type Artifact struct {
	UUID    []string
	Zone    []string
	Title   []string
	service *service.Service
}

// BuilderId returns the unique identifier of this builder
func (*Artifact) BuilderId() string {
	return BuilderId
}

// Destroy destroys the template
func (a *Artifact) Destroy() error {
	log.Printf("Deleting template \"%s\"", a.Title)
	for _, uuid := range a.UUID {
		err := a.service.DeleteStorage(&request.DeleteStorageRequest{
			UUID: uuid,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Files returns the files represented by the artifact
func (*Artifact) Files() []string {
	return nil
}

func (a *Artifact) Id() string {
	sb := strings.Builder{}
	for i, zone := range a.Zone {
		sb.WriteString(fmt.Sprintf("%s:%s\n", zone, a.UUID[i]))
	}
	return sb.String()
}

func (*Artifact) State(name string) interface{} {
	return nil
}

// String returns the string representation of the artifact. It is printed at the end of builds.
func (a *Artifact) String() string {
	return fmt.Sprintf("Private template (UUID: %s, Title: %s, Zone: %s)",
		strings.Join(a.UUID[:], ","),
		strings.Join(a.Title[:], ","),
		strings.Join(a.Zone[:], ","),
	)
}
