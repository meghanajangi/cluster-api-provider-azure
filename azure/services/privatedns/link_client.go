/*
Copyright 2022 The Kubernetes Authors.

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

package privatedns

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
	azureautorest "github.com/Azure/go-autorest/autorest/azure"
	"github.com/pkg/errors"
	infrav1 "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-azure/azure"
	"sigs.k8s.io/cluster-api-provider-azure/util/reconciler"
	"sigs.k8s.io/cluster-api-provider-azure/util/tele"
)

// azureVirtualNetworkLinksClient contains the Azure go-sdk Client for virtual network links.
type azureVirtualNetworkLinksClient struct {
	vnetlinks privatedns.VirtualNetworkLinksClient
}

// newVirtualNetworkLinksClient creates a new virtual network link client.
func newVirtualNetworkLinksClient(auth azure.Authorizer) *azureVirtualNetworkLinksClient {
	linksClient := privatedns.NewVirtualNetworkLinksClientWithBaseURI(auth.BaseURI(), auth.SubscriptionID())
	azure.SetAutoRestClientDefaults(&linksClient.Client, auth.Authorizer())
	return &azureVirtualNetworkLinksClient{
		vnetlinks: linksClient,
	}
}

// CreateOrUpdateAsync creates or updates a virtual network link asynchronously.
// It sends a PUT request to Azure and if accepted without error, the func will return a Future which can be used to track the ongoing
// progress of the operation.
func (avc *azureVirtualNetworkLinksClient) CreateOrUpdateAsync(ctx context.Context, spec azure.ResourceSpecGetter, parameters interface{}) (result interface{}, future azureautorest.FutureAPI, err error) {
	ctx, _, done := tele.StartSpanWithLogger(ctx, "privatedns.azureVirtualNetworkLinksClient.CreateOrUpdateAsync")
	defer done()

	link, ok := parameters.(privatedns.VirtualNetworkLink)
	if !ok {
		return nil, nil, errors.Errorf("%T is not a privatedns.VirtualNetworkLink", parameters)
	}

	createFuture, err := avc.vnetlinks.CreateOrUpdate(ctx, spec.ResourceGroupName(), spec.OwnerResourceName(), spec.ResourceName(), link, "", "")
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultAzureCallTimeout)
	defer cancel()

	err = createFuture.WaitForCompletionRef(ctx, avc.vnetlinks.Client)
	if err != nil {
		// if an error occurs, return the future.
		// this means the long-running operation didn't finish in the specified timeout.
		return nil, &createFuture, err
	}
	result, err = createFuture.Result(avc.vnetlinks)
	// if the operation completed, return a nil future
	return result, nil, err
}

// Get gets the specified virtual network link.
func (avc *azureVirtualNetworkLinksClient) Get(ctx context.Context, spec azure.ResourceSpecGetter) (result interface{}, err error) {
	ctx, _, done := tele.StartSpanWithLogger(ctx, "privatedns.azureVirtualNetworkLinksClient.Get")
	defer done()
	link, err := avc.vnetlinks.Get(ctx, spec.ResourceGroupName(), spec.OwnerResourceName(), spec.ResourceName())
	if err != nil {
		return privatedns.VirtualNetworkLink{}, err
	}
	return link, nil
}

// DeleteAsync deletes a virtual network link asynchronously. DeleteAsync sends a DELETE
// request to Azure and if accepted without error, the func will return a Future which can be used to track the ongoing
// progress of the operation.
func (avc *azureVirtualNetworkLinksClient) DeleteAsync(ctx context.Context, spec azure.ResourceSpecGetter) (future azureautorest.FutureAPI, err error) {
	ctx, _, done := tele.StartSpanWithLogger(ctx, "privatedns.azureVirtualNetworkLinksClient.DeleteAsync")
	defer done()

	deleteFuture, err := avc.vnetlinks.Delete(ctx, spec.ResourceGroupName(), spec.OwnerResourceName(), spec.ResourceName(), "")
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultAzureCallTimeout)
	defer cancel()

	err = deleteFuture.WaitForCompletionRef(ctx, avc.vnetlinks.Client)
	if err != nil {
		// if an error occurs, return the future.
		// this means the long-running operation didn't finish in the specified timeout.
		return &deleteFuture, err
	}
	_, err = deleteFuture.Result(avc.vnetlinks)
	// if the operation completed, return a nil future.
	return nil, err
}

// IsDone returns true if the long-running operation has completed.
func (avc *azureVirtualNetworkLinksClient) IsDone(ctx context.Context, future azureautorest.FutureAPI) (isDone bool, err error) {
	ctx, _, done := tele.StartSpanWithLogger(ctx, "privatedns.azureVirtualNetworkLinksClient.IsDone")
	defer done()

	isDone, err = future.DoneWithContext(ctx, avc.vnetlinks)
	if err != nil {
		return false, errors.Wrap(err, "failed checking if the operation was complete")
	}

	return isDone, nil
}

// Result fetches the result of a long-running operation future.
func (avc *azureVirtualNetworkLinksClient) Result(ctx context.Context, future azureautorest.FutureAPI, futureType string) (result interface{}, err error) {
	_, _, done := tele.StartSpanWithLogger(ctx, "privatedns.azureVirtualNetworkLinksClient.Result")
	defer done()

	if future == nil {
		return nil, errors.Errorf("cannot get result from nil future")
	}

	switch futureType {
	case infrav1.PutFuture:
		// Marshal and Unmarshal the future to put it into the correct future type so we can access the Result function.
		// Unfortunately the FutureAPI can't be casted directly to VirtualNetworkLinksCreateOrUpdateFuture because it is a azureautorest.Future, which doesn't implement the Result function. See PR #1686 for discussion on alternatives.
		// It was converted back to a generic azureautorest.Future from the CAPZ infrav1.Future type stored in Status: https://github.com/kubernetes-sigs/cluster-api-provider-azure/blob/main/azure/converters/futures.go#L49.
		var createFuture *privatedns.VirtualNetworkLinksCreateOrUpdateFuture
		jsonData, err := future.MarshalJSON()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal future")
		}
		if err := json.Unmarshal(jsonData, &createFuture); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal future data")
		}
		return createFuture.Result(avc.vnetlinks)

	case infrav1.DeleteFuture:
		// Delete does not return a result private dns virtual network link.
		return nil, nil

	default:
		return nil, errors.Errorf("unknown future type %q", futureType)
	}
}
