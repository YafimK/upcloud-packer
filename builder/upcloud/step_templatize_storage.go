package upcloud

import (
	"context"
	"fmt"
	"github.com/UpCloudLtd/upcloud-go-sdk/upcloud"
	"github.com/UpCloudLtd/upcloud-go-sdk/upcloud/request"
	"github.com/UpCloudLtd/upcloud-go-sdk/upcloud/service"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
	"golang.org/x/sync/errgroup"
	"time"
)

// StepTemplatizeStorage represents the step that creates a storage template from the newly created server
type StepTemplatizeStorage struct {
}

// Run runs the actual step
func (s *StepTemplatizeStorage) Run(state multistep.StateBag) multistep.StepAction {
	// Store a success indicator in the state
	state.Put("step_templatize_storage_success", false)

	// Extract state
	ui := state.Get("ui").(packer.Ui)
	service := state.Get("service").(service.Service)
	config := state.Get("config").(Config)

	serversDetails := state.Get("server_details").([]*upcloud.ServerDetails)
	templatesDetails := make([]*upcloud.StorageDetails, len(config.Zone))

	ctx, _ := context.WithTimeout(context.Background(), 20*time.Minute)
	errGrp, _ := errgroup.WithContext(ctx)

	for i, serverDetails := range serversDetails {
		i, serverDetails := i, serverDetails
		errGrp.Go(func() error {
			// Stop the server and wait until it has stopped
			ui.Say(fmt.Sprintf("Stopping server \"%s\" ...", serverDetails.Title))
			serverDetails, err := service.StopServer(&request.StopServerRequest{
				UUID: serverDetails.UUID,
			})

			if err != nil {
				return err
			}

			ui.Say(fmt.Sprintf("Waiting for server \"%s\" to enter the \"stopped\" state ...", serverDetails.Title))
			serverDetails, err = service.WaitForServerState(&request.WaitForServerStateRequest{
				UUID:         serverDetails.UUID,
				DesiredState: upcloud.ServerStateStopped,
				Timeout:      config.StateTimeoutDuration,
			})

			if err != nil {
				return err
			}

			ui.Say(fmt.Sprintf("Server \"%s\" is now in \"stopped\" state", serverDetails.Title))

			// Templatize the first disk device in the server
			for _, storage := range serverDetails.StorageDevices {
				if storage.Type == upcloud.StorageTypeDisk {
					ui.Say(fmt.Sprintf("Templatizing storage device \"%s\" ...", storage.Title))

					// Determine the prefix to use for the template title
					prefix := storage.Title
					if config.TemplatePrefix != "" {
						prefix = config.TemplatePrefix
					}

					storageDetails, err := service.TemplatizeStorage(&request.TemplatizeStorageRequest{
						UUID:  storage.UUID,
						Title: fmt.Sprintf("%s-template-%d", prefix, time.Now().Unix()),
					})

					if err != nil {
						return err
					}

					// Wait for the newly templatized storage to enter the "online" state
					ui.Say(fmt.Sprintf("Waiting for storage \"%s\" to enter the \"online\" state", storageDetails.Title))
					storageDetails, err = service.WaitForStorageState(&request.WaitForStorageStateRequest{
						UUID:         storageDetails.UUID,
						DesiredState: upcloud.StorageStateOnline,
						Timeout:      config.StateTimeoutDuration,
					})

					if err != nil {
						return err
					}

					// Storage the details about the templatized storage in the state. Also update our success
					templatesDetails[i] = storageDetails
					// boolean
					return nil
				}
			}
			// No storage found, we'll have to abort
			return fmt.Errorf("Unable to find the storage device to templatize")
		})
	}

	if err := errGrp.Wait(); err != nil {
		return handleError(err, state)
	}
	state.Put("storage_details", templatesDetails)
	state.Put("step_templatize_storage_success", true)
	return multistep.ActionContinue

}

// Cleanup cleans up after the step
func (s *StepTemplatizeStorage) Cleanup(state multistep.StateBag) {
	// Don't perform any cleanup if the step executed successfully
	if state.Get("step_templatize_storage_success").(bool) {
		return
	}

	// Extract state, return if no state has been stored
	if rawDetails, ok := state.GetOk("storage_details"); ok {
		storageDetails := rawDetails.(*upcloud.StorageDetails)

		service := state.Get("service").(service.Service)
		ui := state.Get("ui").(packer.Ui)

		// Delete the storage device
		err := service.DeleteStorage(&request.DeleteStorageRequest{
			UUID: storageDetails.UUID,
		})

		if err != nil {
			ui.Error(fmt.Sprintf("%s", err))
			return
		}
	}
}
