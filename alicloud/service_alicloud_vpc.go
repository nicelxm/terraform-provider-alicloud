package alicloud

import (
	"log"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"

	"fmt"

	"github.com/PaesslerAG/jsonpath"
	util "github.com/alibabacloud-go/tea-utils/service"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/aliyun/terraform-provider-alicloud/alicloud/connectivity"
)

type VpcService struct {
	client *connectivity.AliyunClient
}

func (s *VpcService) DescribeEip(id string) (eip vpc.EipAddress, err error) {

	request := vpc.CreateDescribeEipAddressesRequest()
	request.RegionId = string(s.client.Region)
	request.AllocationId = id
	wait := incrementalWait(3*time.Second, 3*time.Second)
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		raw, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
			return vpcClient.DescribeEipAddresses(request)
		})
		if err != nil {
			if NeedRetry(err) {
				wait()
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		addDebug(request.GetActionName(), raw, request.RpcRequest, request)
		response, _ := raw.(*vpc.DescribeEipAddressesResponse)
		if len(response.EipAddresses.EipAddress) <= 0 || response.EipAddresses.EipAddress[0].AllocationId != id {
			return resource.NonRetryableError(WrapErrorf(Error(GetNotFoundMessage("Eip", id)), NotFoundMsg, ProviderERROR))
		}
		eip = response.EipAddresses.EipAddress[0]
		return nil
	})

	return
}

func (s *VpcService) DescribeEipAssociation(id string) (object vpc.EipAddress, err error) {
	parts, err := ParseResourceId(id, 2)
	if err != nil {
		err = WrapError(err)
		return
	}
	object, err = s.DescribeEip(parts[0])
	if err != nil {
		err = WrapError(err)
		return
	}
	if object.InstanceId != parts[1] {
		err = WrapErrorf(Error(GetNotFoundMessage("Eip Association", id)), NotFoundMsg, ProviderERROR)
	}

	return
}

func (s *VpcService) DescribeNatGateway(id string) (object map[string]interface{}, err error) {
	var response map[string]interface{}
	conn, err := s.client.NewVpcClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "DescribeNatGateways"
	request := map[string]interface{}{
		"RegionId":     s.client.RegionId,
		"NatGatewayId": id,
	}
	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	response, err = conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &runtime)
	if err != nil {
		if IsExpectedErrors(err, []string{"InvalidNatGatewayId.NotFound", "InvalidRegionId.NotFound"}) {
			err = WrapErrorf(Error(GetNotFoundMessage("NatGateway", id)), NotFoundMsg, ProviderERROR)
			return object, err
		}
		err = WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
		return object, err
	}
	addDebug(action, response, request)
	v, err := jsonpath.Get("$.NatGateways.NatGateway", response)
	if err != nil {
		return object, WrapErrorf(err, FailedGetAttributeMsg, id, "$.NatGateways.NatGateway", response)
	}
	if len(v.([]interface{})) < 1 {
		return object, WrapErrorf(Error(GetNotFoundMessage("VPC", id)), NotFoundWithResponse, response)
	} else {
		if v.([]interface{})[0].(map[string]interface{})["NatGatewayId"].(string) != id {
			return object, WrapErrorf(Error(GetNotFoundMessage("VPC", id)), NotFoundWithResponse, response)
		}
	}
	object = v.([]interface{})[0].(map[string]interface{})
	return object, nil
}

func (s *VpcService) DescribeVpc(id string) (object map[string]interface{}, err error) {
	var response map[string]interface{}
	conn, err := s.client.NewVpcClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "DescribeVpcs"
	request := map[string]interface{}{
		"RegionId": s.client.RegionId,
		"VpcId":    id,
	}
	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	response, err = conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &runtime)
	if err != nil {
		if IsExpectedErrors(err, []string{"Forbidden.VpcNotFound", "InvalidVpcID.NotFound"}) {
			err = WrapErrorf(Error(GetNotFoundMessage("Vpc", id)), NotFoundMsg, ProviderERROR)
			return object, err
		}
		err = WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
		return object, err
	}
	addDebug(action, response, request)
	v, err := jsonpath.Get("$.Vpcs.Vpc", response)
	if err != nil {
		return object, WrapErrorf(err, FailedGetAttributeMsg, id, "$.Vpcs.Vpc", response)
	}
	if len(v.([]interface{})) < 1 {
		return object, WrapErrorf(Error(GetNotFoundMessage("VPC", id)), NotFoundWithResponse, response)
	} else {
		if v.([]interface{})[0].(map[string]interface{})["VpcId"].(string) != id {
			return object, WrapErrorf(Error(GetNotFoundMessage("VPC", id)), NotFoundWithResponse, response)
		}
	}
	object = v.([]interface{})[0].(map[string]interface{})
	return object, nil
}

func (s *VpcService) DescribeVpcWithTeadsl(id string) (object map[string]interface{}, err error) {
	conn, err := s.client.NewVpcClient()
	if err != nil {
		return nil, WrapError(err)
	}

	action := "DescribeVpcs"
	request := map[string]interface{}{
		"RegionId": s.client.RegionId,
		"VpcId":    id,
	}
	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	invoker := NewInvoker()
	err = invoker.Run(func() error {
		response, err := conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &runtime)
		if err != nil {
			if IsExpectedErrors(err, []string{"InvalidVpcID.NotFound", "Forbidden.VpcNotFound"}) {
				return WrapErrorf(err, NotFoundMsg, AlibabaCloudSdkGoERROR)
			}
			return WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
		}
		addDebug(action, response, request)

		v, err := jsonpath.Get("$.Vpcs.Vpc", response)
		if err != nil {
			return WrapErrorf(err, FailedGetAttributeMsg, id, "$.Vpcs.Vpc", response)
		}
		if len(v.([]interface{})) < 1 || v.([]interface{})[0].(map[string]interface{})["VpcId"].(string) != id {
			return WrapErrorf(Error(GetNotFoundMessage("VPC", id)), NotFoundWithResponse, response)
		}
		object = v.([]interface{})[0].(map[string]interface{})
		return nil
	})
	return
}

func (s *VpcService) ListTagResources(id string, resourceType string) (object interface{}, err error) {
	conn, err := s.client.NewVpcClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "ListTagResources"
	request := map[string]interface{}{
		"RegionId":     s.client.RegionId,
		"ResourceType": resourceType,
		"ResourceId.1": id,
	}
	tags := make([]interface{}, 0)
	var response map[string]interface{}

	for {
		wait := incrementalWait(3*time.Second, 5*time.Second)
		err = resource.Retry(5*time.Minute, func() *resource.RetryError {
			response, err := conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &util.RuntimeOptions{})
			if err != nil {
				if IsExpectedErrors(err, []string{Throttling}) {
					wait()
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			addDebug(action, response, request)
			v, err := jsonpath.Get("$.TagResources.TagResource", response)
			if err != nil {
				return resource.NonRetryableError(WrapErrorf(err, FailedGetAttributeMsg, id, "$.TagResources.TagResource", response))
			}
			if v != nil {
				tags = append(tags, v.([]interface{})...)
			}
			return nil
		})
		if err != nil {
			err = WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
			return
		}
		if response["NextToken"] == nil {
			break
		}
		request["NextToken"] = response["NextToken"]
	}

	return tags, nil
}

func (s *VpcService) VpcStateRefreshFunc(id string, failStates []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		object, err := s.DescribeVpc(id)
		if err != nil {
			if NotFoundError(err) {
				// Set this to nil as if we didn't find anything.
				return nil, "", nil
			}
			return nil, "", WrapError(err)
		}

		for _, failState := range failStates {
			if object["Status"].(string) == failState {
				return object, object["Status"].(string), WrapError(Error(FailedToReachTargetStatus, object["Status"].(string)))
			}
		}
		return object, object["Status"].(string), nil
	}
}

func (s *VpcService) DescribeVSwitch(id string) (v vpc.DescribeVSwitchAttributesResponse, err error) {
	request := vpc.CreateDescribeVSwitchAttributesRequest()
	request.RegionId = s.client.RegionId
	request.VSwitchId = id

	invoker := NewInvoker()
	err = invoker.Run(func() error {
		raw, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
			return vpcClient.DescribeVSwitchAttributes(request)
		})
		if err != nil {
			if IsExpectedErrors(err, []string{"InvalidVswitchID.NotFound"}) {
				return WrapErrorf(err, NotFoundMsg, AlibabaCloudSdkGoERROR)
			}
			return WrapErrorf(err, DefaultErrorMsg, id, request.GetActionName(), AlibabaCloudSdkGoERROR)
		}
		addDebug(request.GetActionName(), raw, request.RpcRequest, request)
		response, _ := raw.(*vpc.DescribeVSwitchAttributesResponse)
		if response.VSwitchId != id {
			return WrapErrorf(Error(GetNotFoundMessage("vswitch", id)), NotFoundMsg, ProviderERROR)
		}
		v = *response
		return nil
	})
	return
}

func (s *VpcService) DescribeVSwitchWithTeadsl(id string) (object map[string]interface{}, err error) {
	conn, err := s.client.NewVpcClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "DescribeVSwitchAttributes"
	request := map[string]interface{}{
		"RegionId":  s.client.RegionId,
		"VSwitchId": id,
	}

	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	response, err := conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &runtime)
	if err != nil {
		if IsExpectedErrors(err, []string{"InvalidVswitchID.NotFound"}) {
			return nil, WrapErrorf(err, NotFoundMsg, AlibabaCloudSdkGoERROR)
		}
		return nil, WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
	}
	addDebug(action, response, request)
	if v, ok := response["VSwitchId"].(string); ok && v != id {
		return nil, WrapErrorf(Error(GetNotFoundMessage("vswitch", id)), NotFoundMsg, ProviderERROR)
	}
	return response, nil
}

func (s *VpcService) VSwitchStateRefreshFunc(id string, failStates []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		object, err := s.DescribeVSwitch(id)
		if err != nil {
			if NotFoundError(err) {
				// Set this to nil as if we didn't find anything.
				return nil, "", nil
			}
			return nil, "", WrapError(err)
		}

		for _, failState := range failStates {
			if object.Status == failState {
				return object, object.Status, WrapError(Error(FailedToReachTargetStatus, object.Status))
			}
		}

		return object, object.Status, nil
	}
}

func (s *VpcService) DescribeSnatEntry(id string) (object map[string]interface{}, err error) {
	var response map[string]interface{}
	conn, err := s.client.NewVpcClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "DescribeSnatTableEntries"
	parts, err := ParseResourceId(id, 2)
	if err != nil {
		err = WrapError(err)
		return
	}
	request := map[string]interface{}{
		"RegionId":    s.client.RegionId,
		"SnatEntryId": parts[1],
		"SnatTableId": parts[0],
	}
	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	response, err = conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &runtime)
	if err != nil {
		if IsExpectedErrors(err, []string{"InvalidRegionId.NotFound", "InvalidSnatEntryId.NotFound", "InvalidSnatTableId.NotFound"}) {
			err = WrapErrorf(Error(GetNotFoundMessage("SnatEntry", id)), NotFoundMsg, ProviderERROR)
			return object, err
		}
		err = WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
		return object, err
	}
	addDebug(action, response, request)
	v, err := jsonpath.Get("$.SnatTableEntries.SnatTableEntry", response)
	if err != nil {
		return object, WrapErrorf(err, FailedGetAttributeMsg, id, "$.SnatTableEntries.SnatTableEntry", response)
	}
	if len(v.([]interface{})) < 1 {
		return object, WrapErrorf(Error(GetNotFoundMessage("VPC", id)), NotFoundWithResponse, response)
	} else {
		if v.([]interface{})[0].(map[string]interface{})["SnatEntryId"].(string) != parts[1] {
			return object, WrapErrorf(Error(GetNotFoundMessage("VPC", id)), NotFoundWithResponse, response)
		}
	}
	object = v.([]interface{})[0].(map[string]interface{})
	return object, nil
}

func (s *VpcService) DescribeForwardEntry(id string) (object map[string]interface{}, err error) {
	var response map[string]interface{}
	conn, err := s.client.NewVpcClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "DescribeForwardTableEntries"
	parts, err := ParseResourceId(id, 2)
	if err != nil {
		err = WrapError(err)
		return
	}
	request := map[string]interface{}{
		"RegionId":       s.client.RegionId,
		"ForwardEntryId": parts[1],
		"ForwardTableId": parts[0],
	}
	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	response, err = conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &runtime)
	if err != nil {
		if IsExpectedErrors(err, []string{"InvalidForwardEntryId.NotFound", "InvalidForwardTableId.NotFound", "InvalidRegionId.NotFound"}) {
			err = WrapErrorf(Error(GetNotFoundMessage("ForwardEntry", id)), NotFoundMsg, ProviderERROR)
			return object, err
		}
		err = WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
		return object, err
	}
	addDebug(action, response, request)
	v, err := jsonpath.Get("$.ForwardTableEntries.ForwardTableEntry", response)
	if err != nil {
		return object, WrapErrorf(err, FailedGetAttributeMsg, id, "$.ForwardTableEntries.ForwardTableEntry", response)
	}
	if len(v.([]interface{})) < 1 {
		return object, WrapErrorf(Error(GetNotFoundMessage("VPC", id)), NotFoundWithResponse, response)
	} else {
		if v.([]interface{})[0].(map[string]interface{})["ForwardEntryId"].(string) != parts[1] {
			return object, WrapErrorf(Error(GetNotFoundMessage("VPC", id)), NotFoundWithResponse, response)
		}
	}
	object = v.([]interface{})[0].(map[string]interface{})
	return object, nil
}

func (s *VpcService) QueryRouteTableById(routeTableId string) (rt vpc.RouteTable, err error) {
	request := vpc.CreateDescribeRouteTablesRequest()
	request.RegionId = s.client.RegionId
	request.RouteTableId = routeTableId

	invoker := NewInvoker()
	err = invoker.Run(func() error {
		raw, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
			return vpcClient.DescribeRouteTables(request)
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, routeTableId, request.GetActionName(), AlibabaCloudSdkGoERROR)
		}
		addDebug(request.GetActionName(), raw, request.RpcRequest, request)
		response, _ := raw.(*vpc.DescribeRouteTablesResponse)
		if len(response.RouteTables.RouteTable) == 0 ||
			response.RouteTables.RouteTable[0].RouteTableId != routeTableId {
			return WrapErrorf(Error(GetNotFoundMessage("RouteTable", routeTableId)), NotFoundMsg, ProviderERROR)
		}
		rt = response.RouteTables.RouteTable[0]
		return nil
	})
	return
}

func (s *VpcService) DescribeRouteEntry(id string) (*vpc.RouteEntry, error) {
	v := &vpc.RouteEntry{}
	parts, err := ParseResourceId(id, 5)
	if err != nil {
		return v, WrapError(err)
	}
	rtId, cidr, nexthop_type, nexthop_id := parts[0], parts[2], parts[3], parts[4]

	request := vpc.CreateDescribeRouteTablesRequest()
	request.RegionId = s.client.RegionId
	request.RouteTableId = rtId

	invoker := NewInvoker()
	for {
		var raw interface{}
		if err := invoker.Run(func() error {
			response, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
				return vpcClient.DescribeRouteTables(request)
			})
			raw = response
			return err
		}); err != nil {
			return v, WrapErrorf(err, DefaultErrorMsg, id, request.GetActionName(), AlibabaCloudSdkGoERROR)
		}
		addDebug(request.GetActionName(), raw, request.RpcRequest, request)
		response, _ := raw.(*vpc.DescribeRouteTablesResponse)
		if len(response.RouteTables.RouteTable) < 1 {
			break
		}
		for _, table := range response.RouteTables.RouteTable {
			for _, entry := range table.RouteEntrys.RouteEntry {
				if entry.DestinationCidrBlock == cidr && entry.NextHopType == nexthop_type && entry.InstanceId == nexthop_id {
					return &entry, nil
				}
			}
		}
		if len(response.RouteTables.RouteTable) < PageSizeLarge {
			break
		}

		if page, err := getNextpageNumber(request.PageNumber); err != nil {
			return v, WrapError(err)
		} else {
			request.PageNumber = page
		}
	}

	return v, WrapErrorf(Error(GetNotFoundMessage("RouteEntry", id)), NotFoundMsg, ProviderERROR)
}

func (s *VpcService) DescribeRouterInterface(id, regionId string) (ri vpc.RouterInterfaceType, err error) {
	request := vpc.CreateDescribeRouterInterfacesRequest()
	request.RegionId = regionId
	values := []string{id}
	filter := []vpc.DescribeRouterInterfacesFilter{
		{
			Key:   "RouterInterfaceId",
			Value: &values,
		},
	}
	request.Filter = &filter
	invoker := NewInvoker()
	err = invoker.Run(func() error {
		raw, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
			return vpcClient.DescribeRouterInterfaces(request)
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, id, request.GetActionName(), AlibabaCloudSdkGoERROR)
		}
		addDebug(request.GetActionName(), raw, request.RpcRequest, request)
		response, _ := raw.(*vpc.DescribeRouterInterfacesResponse)
		if len(response.RouterInterfaceSet.RouterInterfaceType) <= 0 ||
			response.RouterInterfaceSet.RouterInterfaceType[0].RouterInterfaceId != id {
			return WrapErrorf(Error(GetNotFoundMessage("RouterInterface", id)), NotFoundMsg, ProviderERROR)
		}
		ri = response.RouterInterfaceSet.RouterInterfaceType[0]
		return nil
	})
	return
}

func (s *VpcService) DescribeRouterInterfaceConnection(id, regionId string) (ri vpc.RouterInterfaceType, err error) {
	ri, err = s.DescribeRouterInterface(id, regionId)
	if err != nil {
		return ri, WrapError(err)
	}

	if ri.OppositeInterfaceId == "" || ri.OppositeRouterType == "" ||
		ri.OppositeRouterId == "" || ri.OppositeInterfaceOwnerId == "" {
		return ri, WrapErrorf(Error(GetNotFoundMessage("RouterInterface", id)), NotFoundMsg, ProviderERROR)
	}
	return ri, nil
}

func (s *VpcService) DescribeCenInstanceGrant(id string) (rule vpc.CbnGrantRule, err error) {
	request := vpc.CreateDescribeGrantRulesToCenRequest()
	parts, err := ParseResourceId(id, 3)
	if err != nil {
		return rule, WrapError(err)
	}
	cenId := parts[0]
	instanceId := parts[1]
	instanceType, err := GetCenChildInstanceType(instanceId)
	if err != nil {
		return rule, WrapError(err)
	}

	request.RegionId = s.client.RegionId
	request.InstanceId = instanceId
	request.InstanceType = instanceType

	invoker := NewInvoker()
	err = invoker.Run(func() error {
		raw, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
			return vpcClient.DescribeGrantRulesToCen(request)
		})
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, id, request.GetActionName(), AlibabaCloudSdkGoERROR)
		}
		addDebug(request.GetActionName(), raw, request.RpcRequest, request)
		response, _ := raw.(*vpc.DescribeGrantRulesToCenResponse)
		ruleList := response.CenGrantRules.CbnGrantRule
		if len(ruleList) <= 0 {
			return WrapErrorf(Error(GetNotFoundMessage("GrantRules", id)), NotFoundMsg, ProviderERROR)
		}

		for ruleNum := 0; ruleNum <= len(response.CenGrantRules.CbnGrantRule)-1; ruleNum++ {
			if ruleList[ruleNum].CenInstanceId == cenId {
				rule = ruleList[ruleNum]
				return nil
			}
		}

		return WrapErrorf(Error(GetNotFoundMessage("GrantRules", id)), NotFoundMsg, ProviderERROR)
	})
	return
}

func (s *VpcService) WaitForCenInstanceGrant(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	parts, err := ParseResourceId(id, 3)
	if err != nil {
		return WrapError(err)
	}
	instanceId := parts[1]
	ownerId := parts[2]
	for {
		object, err := s.DescribeCenInstanceGrant(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		if object.CenInstanceId == instanceId && fmt.Sprint(object.CenOwnerId) == ownerId && status != Deleted {
			break
		}
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, object.CenInstanceId, instanceId, ProviderERROR)
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
	return nil
}

func (s *VpcService) DescribeCommonBandwidthPackage(id string) (object map[string]interface{}, err error) {
	var response map[string]interface{}
	conn, err := s.client.NewVpcClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "DescribeCommonBandwidthPackages"
	request := map[string]interface{}{
		"RegionId":           s.client.RegionId,
		"BandwidthPackageId": id,
	}
	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	response, err = conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &runtime)
	if err != nil {
		if IsExpectedErrors(err, []string{"InvalidRegionId.NotFound"}) {
			err = WrapErrorf(Error(GetNotFoundMessage("CommonBandwidthPackage", id)), NotFoundMsg, ProviderERROR)
			return object, err
		}
		err = WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
		return object, err
	}
	addDebug(action, response, request)
	v, err := jsonpath.Get("$.CommonBandwidthPackages.CommonBandwidthPackage", response)
	if err != nil {
		return object, WrapErrorf(err, FailedGetAttributeMsg, id, "$.CommonBandwidthPackages.CommonBandwidthPackage", response)
	}
	if len(v.([]interface{})) < 1 {
		return object, WrapErrorf(Error(GetNotFoundMessage("VPC", id)), NotFoundWithResponse, response)
	} else {
		if v.([]interface{})[0].(map[string]interface{})["BandwidthPackageId"].(string) != id {
			return object, WrapErrorf(Error(GetNotFoundMessage("VPC", id)), NotFoundWithResponse, response)
		}
	}
	object = v.([]interface{})[0].(map[string]interface{})
	return object, nil
}

func (s *VpcService) DescribeCommonBandwidthPackageAttachment(id string) (object map[string]interface{}, err error) {
	parts, err := ParseResourceId(id, 2)
	if err != nil {
		return object, WrapError(err)
	}
	bandwidthPackageId, ipInstanceId := parts[0], parts[1]

	object, err = s.DescribeCommonBandwidthPackage(bandwidthPackageId)
	if err != nil {
		return object, WrapError(err)
	}

	if val, ok := object["PublicIpAddresses"].(map[string]interface{}); ok {
		if vs, ok := val["PublicIpAddresse"]; ok {
			for _, ipAddresse := range vs.([]interface{}) {
				item := ipAddresse.(map[string]interface{})
				if fmt.Sprint(item["AllocationId"]) == ipInstanceId {
					return object, nil
				}
			}
		}
	}
	return object, WrapErrorf(Error(GetNotFoundMessage("CommonBandWidthPackageAttachment", id)), NotFoundMsg, ProviderERROR)
}

func (s *VpcService) DescribeRouteTable(id string) (object map[string]interface{}, err error) {
	var response map[string]interface{}
	conn, err := s.client.NewVpcClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "DescribeRouteTableList"
	request := map[string]interface{}{
		"RegionId":     s.client.RegionId,
		"RouteTableId": id,
	}
	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	response, err = conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &runtime)
	if err != nil {
		err = WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
		return
	}
	addDebug(action, response, request)
	if fmt.Sprintf(`%v`, response["Code"]) != "200" {
		err = WrapErrorf(err, ResponseCodeMsg, id, action, response)
		return object, err
	}
	v, err := jsonpath.Get("$.RouterTableList.RouterTableListType", response)
	if err != nil {
		return object, WrapErrorf(err, FailedGetAttributeMsg, id, "$.RouterTableList.RouterTableListType", response)
	}
	if len(v.([]interface{})) < 1 {
		return object, WrapErrorf(Error(GetNotFoundMessage("VPC", id)), NotFoundWithResponse, response)
	} else {
		if v.([]interface{})[0].(map[string]interface{})["RouteTableId"].(string) != id {
			return object, WrapErrorf(Error(GetNotFoundMessage("VPC", id)), NotFoundWithResponse, response)
		}
	}
	object = v.([]interface{})[0].(map[string]interface{})
	return object, nil
}

func (s *VpcService) DescribeRouteTableAttachment(id string) (v map[string]interface{}, err error) {
	parts, err := ParseResourceId(id, 2)
	if err != nil {
		return v, WrapError(err)
	}
	invoker := NewInvoker()
	routeTableId := parts[0]
	vSwitchId := parts[1]

	err = invoker.Run(func() error {
		object, err := s.DescribeRouteTable(routeTableId)
		if err != nil {
			return WrapError(err)
		}

		if val, ok := object["VSwitchIds"].(map[string]interface{}); ok {
			if vs, ok := val["VSwitchId"]; ok {
				for _, id := range vs.([]interface{}) {
					if fmt.Sprint(id) == vSwitchId {
						v = object
						return nil
					}
				}
			}
		}

		return WrapErrorf(Error(GetNotFoundMessage("RouteTableAttachment", id)), NotFoundMsg, ProviderERROR)
	})
	return v, WrapError(err)
}

func (s *VpcService) WaitForVSwitch(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		object, err := s.DescribeVSwitch(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		if object.Status == string(status) {
			return nil
		}
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, object.Status, string(status), ProviderERROR)
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
}

func (s *VpcService) WaitForRouteEntry(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		object, err := s.DescribeRouteEntry(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		if object.Status == string(status) {
			return nil
		}
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, object.Status, status, ProviderERROR)
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
}

func (s *VpcService) WaitForAllRouteEntriesAvailable(routeTableId string, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		table, err := s.QueryRouteTableById(routeTableId)
		if err != nil {
			return WrapError(err)
		}
		success := true
		for _, routeEntry := range table.RouteEntrys.RouteEntry {
			if routeEntry.Status != string(Available) {
				success = false
				break
			}
		}
		if success {
			break
		}
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, routeTableId, GetFunc(1), timeout, Available, Null, ProviderERROR)
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
	return nil
}

func (s *VpcService) WaitForRouterInterface(id, regionId string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		object, err := s.DescribeRouterInterface(id, regionId)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		if object.Status == string(status) {
			return nil
		}
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, object.Status, string(status), ProviderERROR)
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
}

func (s *VpcService) WaitForRouterInterfaceConnection(id, regionId string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		object, err := s.DescribeRouterInterfaceConnection(id, regionId)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		if object.Status == string(status) {
			return nil
		}
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, object.Status, string(status), ProviderERROR)
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
}

func (s *VpcService) WaitForEip(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		object, err := s.DescribeEip(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		if object.Status == string(status) {
			return nil
		}
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, object.Status, string(status), ProviderERROR)
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
}

func (s *VpcService) WaitForEipAssociation(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		object, err := s.DescribeEipAssociation(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		if object.Status == string(status) {
			return nil
		}
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, object.Status, string(status), ProviderERROR)
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
}

func (s *VpcService) DeactivateRouterInterface(interfaceId string) error {
	request := vpc.CreateDeactivateRouterInterfaceRequest()
	request.RegionId = s.client.RegionId
	request.RouterInterfaceId = interfaceId
	raw, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
		return vpcClient.DeactivateRouterInterface(request)
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, "RouterInterface", request.GetActionName(), AlibabaCloudSdkGoERROR)
	}
	addDebug(request.GetActionName(), raw, request.RpcRequest, request)
	return nil
}

func (s *VpcService) ActivateRouterInterface(interfaceId string) error {
	request := vpc.CreateActivateRouterInterfaceRequest()
	request.RegionId = s.client.RegionId
	request.RouterInterfaceId = interfaceId
	raw, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
		return vpcClient.ActivateRouterInterface(request)
	})
	if err != nil {
		return WrapErrorf(err, DefaultErrorMsg, "RouterInterface", request.GetActionName(), AlibabaCloudSdkGoERROR)
	}
	addDebug(request.GetActionName(), raw, request.RpcRequest, request)
	return nil
}

func (s *VpcService) WaitForCommonBandwidthPackageAttachment(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		object, err := s.DescribeCommonBandwidthPackageAttachment(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}

		if fmt.Sprint(object["Status"]) == string(status) {
			return nil
		}
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, fmt.Sprint(object["Status"]), string(status), ProviderERROR)
		}
	}
}

// Flattens an array of vpc.public_ip_addresses into a []map[string]string
func (s *VpcService) FlattenPublicIpAddressesMappings(list []vpc.PublicIpAddresse) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(list))

	for _, i := range list {
		l := map[string]interface{}{
			"ip_address":    i.IpAddress,
			"allocation_id": i.AllocationId,
		}
		result = append(result, l)
	}

	return result
}

func (s *VpcService) WaitForRouteTableAttachment(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		object, err := s.DescribeRouteTableAttachment(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}

		if fmt.Sprint(object["Status"]) == string(status) {
			return nil
		}

		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, fmt.Sprint(object["Status"]), string(status), ProviderERROR)
		}
		time.Sleep(3 * time.Second)
	}
}

func (s *VpcService) DescribeNetworkAcl(id string) (networkAcl vpc.NetworkAcl, err error) {

	request := vpc.CreateDescribeNetworkAclsRequest()
	request.RegionId = s.client.RegionId
	request.NetworkAclId = id

	raw, err := s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
		return vpcClient.DescribeNetworkAcls(request)
	})
	if err != nil {
		if IsExpectedErrors(err, []string{"InvalidNetworkAcl.NotFound"}) {
			return networkAcl, WrapErrorf(err, NotFoundMsg, AlibabaCloudSdkGoERROR)
		}
		return networkAcl, WrapErrorf(err, DefaultErrorMsg, id, request.GetActionName(), AlibabaCloudSdkGoERROR)
	}
	response, _ := raw.(*vpc.DescribeNetworkAclsResponse)
	addDebug(request.GetActionName(), raw, request.RpcRequest, request)
	if len(response.NetworkAcls.NetworkAcl) <= 0 || response.NetworkAcls.NetworkAcl[0].NetworkAclId != id {
		return networkAcl, WrapErrorf(Error(GetNotFoundMessage("NetworkAcl", id)), NotFoundMsg, ProviderERROR)
	}
	return response.NetworkAcls.NetworkAcl[0], nil
}

func (s *VpcService) DescribeNetworkAclAttachment(id string, resource []vpc.Resource) (err error) {

	invoker := NewInvoker()
	return invoker.Run(func() error {
		object, err := s.DescribeNetworkAcl(id)
		if err != nil {
			return WrapError(err)
		}
		if len(object.Resources.Resource) < 1 {
			return WrapErrorf(Error(GetNotFoundMessage("Network Acl Attachment", id)), NotFoundMsg, ProviderERROR)
		}
		success := true
		for _, source := range object.Resources.Resource {
			success = false
			for _, res := range resource {
				if source.ResourceId == res.ResourceId {
					success = true
				}
			}
			if success == false {
				return WrapErrorf(Error(GetNotFoundMessage("Network Acl Attachment", id)), NotFoundMsg, ProviderERROR)
			}
		}
		return nil
	})
}

func (s *VpcService) WaitForNetworkAcl(networkAclId string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		object, err := s.DescribeNetworkAcl(networkAclId)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		success := true
		// Check Acl's binding resources
		for _, resource := range object.Resources.Resource {
			if resource.Status != string(BINDED) {
				success = false
			}
		}
		if object.Status == string(status) && success == true {
			return nil
		}
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, networkAclId, GetFunc(1), timeout, object.Status, string(status), ProviderERROR)
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
}

func (s *VpcService) WaitForNetworkAclAttachment(id string, resource []vpc.Resource, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		err := s.DescribeNetworkAclAttachment(id, resource)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		object, err := s.DescribeNetworkAcl(id)
		success := true
		// Check Acl's binding resources
		for _, resource := range object.Resources.Resource {
			if resource.Status != string(BINDED) {
				success = false
			}
		}
		if object.Status == string(status) && success == true {
			return nil
		}
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, object.Status, string(status), ProviderERROR)
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
}

func (s *VpcService) DescribeTags(resourceId string, resourceTags map[string]interface{}, resourceType TagResourceType) (tags []vpc.TagResource, err error) {
	request := vpc.CreateListTagResourcesRequest()
	request.RegionId = s.client.RegionId
	request.ResourceType = string(resourceType)
	request.ResourceId = &[]string{resourceId}
	if resourceTags != nil && len(resourceTags) > 0 {
		var reqTags []vpc.ListTagResourcesTag
		for key, value := range resourceTags {
			reqTags = append(reqTags, vpc.ListTagResourcesTag{
				Key:   key,
				Value: value.(string),
			})
		}
		request.Tag = &reqTags
	}

	wait := incrementalWait(3*time.Second, 5*time.Second)
	var raw interface{}

	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		raw, err = s.client.WithVpcClient(func(vpcClient *vpc.Client) (interface{}, error) {
			return vpcClient.ListTagResources(request)
		})
		if err != nil {
			if IsExpectedErrors(err, []string{Throttling}) {
				wait()
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		addDebug(request.GetActionName(), raw, request.RpcRequest, request)
		return nil
	})
	if err != nil {
		err = WrapErrorf(err, DefaultErrorMsg, resourceId, request.GetActionName(), AlibabaCloudSdkGoERROR)
		return
	}
	response, _ := raw.(*vpc.ListTagResourcesResponse)

	return response.TagResources.TagResource, nil
}

func (s *VpcService) setInstanceTags(d *schema.ResourceData, resourceType TagResourceType) error {
	if d.HasChange("tags") {
		added, removed := parsingTags(d)
		conn, err := s.client.NewVpcClient()
		if err != nil {
			return WrapError(err)
		}

		if len(removed) > 0 {
			action := "UnTagResources"
			request := map[string]interface{}{
				"RegionId":     s.client.RegionId,
				"ResourceId.1": d.Id(),
				"ResourceType": string(resourceType),
			}
			for i, key := range removed {
				request[fmt.Sprintf("TagKey.%d", i+1)] = key
			}

			wait := incrementalWait(2*time.Second, 1*time.Second)
			err := resource.Retry(10*time.Minute, func() *resource.RetryError {
				response, err := conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &util.RuntimeOptions{})
				if err != nil {
					if IsThrottling(err) {
						wait()
						return resource.RetryableError(err)

					}
					return resource.NonRetryableError(err)
				}
				addDebug(action, response, request)
				return nil
			})
			if err != nil {
				return WrapErrorf(err, DefaultErrorMsg, d.Id(), action, AlibabaCloudSdkGoERROR)
			}
		}

		if len(added) > 0 {
			action := "TagResources"
			request := map[string]interface{}{
				"RegionId":     s.client.RegionId,
				"ResourceId.1": d.Id(),
				"ResourceType": string(resourceType),
			}
			count := 1
			for key, value := range added {
				request[fmt.Sprintf("Tag.%d.Key", count)] = key
				request[fmt.Sprintf("Tag.%d.Value	", count)] = value
				count++
			}

			wait := incrementalWait(2*time.Second, 1*time.Second)
			err := resource.Retry(10*time.Minute, func() *resource.RetryError {
				response, err := conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &util.RuntimeOptions{})
				if err != nil {
					if IsThrottling(err) {
						wait()
						return resource.RetryableError(err)

					}
					return resource.NonRetryableError(err)
				}
				addDebug(action, response, request)
				return nil
			})
			if err != nil {
				return WrapErrorf(err, DefaultErrorMsg, d.Id(), action, AlibabaCloudSdkGoERROR)
			}
		}

		d.SetPartial("tags")
	}

	return nil
}

func (s *VpcService) tagsToMap(tags []vpc.TagResource) map[string]string {
	result := make(map[string]string)
	for _, t := range tags {
		if !s.ignoreTag(t) {
			result[t.TagKey] = t.TagValue
		}
	}
	return result
}

func (s *VpcService) ignoreTag(t vpc.TagResource) bool {
	filter := []string{"^aliyun", "^acs:", "^http://", "^https://"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching prefix %v with %v\n", v, t.TagKey)
		ok, _ := regexp.MatchString(v, t.TagKey)
		if ok {
			log.Printf("[DEBUG] Found Alibaba Cloud specific t %s (val: %s), ignoring.\n", t.TagKey, t.TagValue)
			return true
		}
	}
	return false
}

func (s *VpcService) setInstanceSecondaryCidrBlocks(d *schema.ResourceData) error {
	if d.HasChange("secondary_cidr_blocks") {
		oraw, nraw := d.GetChange("secondary_cidr_blocks")
		removed := oraw.([]interface{})
		added := nraw.([]interface{})
		conn, err := s.client.NewVpcClient()
		if err != nil {
			return WrapError(err)
		}
		if len(removed) > 0 {
			action := "UnassociateVpcCidrBlock"
			request := map[string]interface{}{
				"RegionId": s.client.RegionId,
				"VpcId":    d.Id(),
			}
			for _, item := range removed {
				request["SecondaryCidrBlock"] = item
				response, err := conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &util.RuntimeOptions{})
				if err != nil {
					return WrapErrorf(err, DefaultErrorMsg, d.Id(), action, AlibabaCloudSdkGoERROR)
				}
				addDebug(action, response, request)
			}
		}

		if len(added) > 0 {
			action := "AssociateVpcCidrBlock"
			request := map[string]interface{}{
				"RegionId": s.client.RegionId,
				"VpcId":    d.Id(),
			}
			for _, item := range added {
				request["SecondaryCidrBlock"] = item
				response, err := conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &util.RuntimeOptions{})
				if err != nil {
					return WrapErrorf(err, DefaultErrorMsg, d.Id(), action, AlibabaCloudSdkGoERROR)
				}
				addDebug(action, response, request)
			}
		}
		d.SetPartial("secondary_cidr_blocks")
	}
	return nil
}

func (s *VpcService) DescribeNatGatewayTransform(id string) ([]interface{}, error) {
	conn, err := s.client.NewVpcClient()
	if err != nil {
		return nil, WrapError(err)
	}

	action := "GetNatGatewayConvertStatus"
	request := map[string]interface{}{
		"RegionId":     s.client.RegionId,
		"NatGatewayId": id,
	}
	request["ClientToken"] = buildClientToken("GetNatGatewayConvertStatus")

	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)

	response, err1 := conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &runtime)
	if err1 != nil {
		if IsExpectedErrors(err1, []string{"InvalidVpcID.NotFound", "Forbidden.VpcNotFound"}) {
			return nil, WrapErrorf(err, NotFoundMsg, AlibabaCloudSdkGoERROR)
		}
		return nil, WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
	}
	addDebug(action, response, request)

	ob, err2 := jsonpath.Get("$.ConvertSteps", response)
	if err2 != nil {
		return nil, WrapErrorf(err2, FailedGetAttributeMsg, id, "$.ConvertSteps", response)
	}

	natType, err3 := jsonpath.Get("$.DstNatType", response)
	if err3 != nil {
		return nil, WrapErrorf(err3, FailedGetAttributeMsg, id, "$.DstNatType", response)
	}

	object := ob.([]interface{})
	if len(object) < 1 || natType.(string) != "Enhanced" {
		return nil, WrapErrorf(Error(GetNotFoundMessage("NAT", id)), NotFoundWithResponse, response)
	}
	return object, nil

}

func (s *VpcService) WaitForNatGatewayTransform(id string, timeout int64) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		object, err := s.DescribeNatGatewayTransform(id)
		if err != nil {
			if NotFoundError(err) {
				return err
			}
			if IsExpectedErrors(err, []string{"OperationFailed.NatGwRouteInMiddleStatusError"}) {
				return nil
			}
			return err
		}

		isOk := false
		for _, v := range object {
			val := v.(map[string]interface{})
			if val["StepName"].(string) == "end_success" && val["StepStatus"].(string) == "successful" {
				isOk = true
				break
			}
		}
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, "", "", ProviderERROR)
		}
		if isOk {
			break
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
	return nil
}

func (s *VpcService) DescribeFlowLogs(id string) (map[string]interface{}, error) {
	conn, err := s.client.NewVpcClient()
	if err != nil {
		return nil, WrapError(err)
	}

	action := "DescribeFlowLogs"
	request := map[string]interface{}{
		"RegionId":  s.client.RegionId,
		"FlowLogId": id,
	}

	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)

	response, err1 := conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &runtime)
	if err1 != nil {
		return nil, WrapErrorf(err1, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
	}
	addDebug(action, response, request)

	ob, err2 := jsonpath.Get("$.FlowLogs.FlowLog", response)
	if err2 != nil {
		return nil, WrapErrorf(Error(GetNotFoundMessage("vpc flow log", id)), NotFoundWithResponse, response)
	}
	object := ob.([]interface{})
	for _, v := range object {
		val := v.(map[string]interface{})
		if val["FlowLogId"] == id {
			return val, nil
		}
	}
	return nil, WrapErrorf(Error(GetNotFoundMessage("vpc flow log", id)), NotFoundWithResponse, response)
}

func (s *VpcService) WaitForVpcFlowLog(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)

	for {
		object, err := s.DescribeFlowLogs(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			}
			return WrapError(err)
		}

		if object["Status"].(string) == string(status) {
			return nil
		}
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, fmt.Sprintf("%v", object["Status"].(string)), string(status), ProviderERROR)
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
}

func (s *VpcService) DescribeRouteTableList(id string) (object map[string]interface{}, err error) {
	var response map[string]interface{}
	conn, err := s.client.NewVpcClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "DescribeRouteTableList"
	request := map[string]interface{}{
		"RegionId": s.client.RegionId,
		"VpcId":    id,
	}
	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	response, err = conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &runtime)
	if err != nil {
		err = WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
		return
	}
	addDebug(action, response, request)
	if fmt.Sprintf(`%v`, response["Code"]) != "200" {
		err = Error("DescribeRouteTableList failed for " + response["Message"].(string))
		return object, err
	}
	v, err := jsonpath.Get("$.RouterTableList.RouterTableListType", response)
	if err != nil {
		return object, WrapErrorf(err, FailedGetAttributeMsg, id, "$.RouterTableList.RouterTableListType", response)
	}
	if len(v.([]interface{})) < 1 {
		return object, WrapErrorf(Error(GetNotFoundMessage("VPC", id)), NotFoundWithResponse, response)
	} else {
		if v.([]interface{})[0].(map[string]interface{})["VpcId"].(string) != id {
			return object, WrapErrorf(Error(GetNotFoundMessage("VPC", id)), NotFoundWithResponse, response)
		}
	}
	object = v.([]interface{})[0].(map[string]interface{})
	return object, nil
}

func (s *VpcService) SetResourceTags(d *schema.ResourceData, resourceType string) error {

	if d.HasChange("tags") {
		added, removed := parsingTags(d)
		conn, err := s.client.NewVpcClient()
		if err != nil {
			return WrapError(err)
		}

		removedTagKeys := make([]string, 0)
		for _, v := range removed {
			if !ignoredTags(v, "") {
				removedTagKeys = append(removedTagKeys, v)
			}
		}
		if len(removedTagKeys) > 0 {
			action := "UnTagResources"
			request := map[string]interface{}{
				"RegionId":     s.client.RegionId,
				"ResourceType": resourceType,
				"ResourceId.1": d.Id(),
			}
			for i, key := range removedTagKeys {
				request[fmt.Sprintf("TagKey.%d", i+1)] = key
			}
			wait := incrementalWait(2*time.Second, 1*time.Second)
			err := resource.Retry(10*time.Minute, func() *resource.RetryError {
				response, err := conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &util.RuntimeOptions{})
				if err != nil {
					if IsThrottling(err) {
						wait()
						return resource.RetryableError(err)

					}
					return resource.NonRetryableError(err)
				}
				addDebug(action, response, request)
				return nil
			})
			if err != nil {
				return WrapErrorf(err, DefaultErrorMsg, d.Id(), action, AlibabaCloudSdkGoERROR)
			}
		}
		if len(added) > 0 {
			action := "TagResources"
			request := map[string]interface{}{
				"RegionId":     s.client.RegionId,
				"ResourceType": resourceType,
				"ResourceId.1": d.Id(),
			}
			count := 1
			for key, value := range added {
				request[fmt.Sprintf("Tag.%d.Key", count)] = key
				request[fmt.Sprintf("Tag.%d.Value", count)] = value
				count++
			}

			wait := incrementalWait(2*time.Second, 1*time.Second)
			err := resource.Retry(10*time.Minute, func() *resource.RetryError {
				response, err := conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &util.RuntimeOptions{})
				if err != nil {
					if IsThrottling(err) {
						wait()
						return resource.RetryableError(err)

					}
					return resource.NonRetryableError(err)
				}
				addDebug(action, response, request)
				return nil
			})
			if err != nil {
				return WrapErrorf(err, DefaultErrorMsg, d.Id(), action, AlibabaCloudSdkGoERROR)
			}
		}
		d.SetPartial("tags")
	}
	return nil
}

func (s *VpcService) DescribeVswitch(id string) (object map[string]interface{}, err error) {
	var response map[string]interface{}
	conn, err := s.client.NewVpcClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "DescribeVSwitchAttributes"
	request := map[string]interface{}{
		"RegionId":  s.client.RegionId,
		"VSwitchId": id,
	}
	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	response, err = conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &runtime)
	if err != nil {
		if IsExpectedErrors(err, []string{"InvalidVSwitchId.NotFound", "InvalidVswitchID.NotFound"}) {
			err = WrapErrorf(Error(GetNotFoundMessage("Vswitch", id)), NotFoundMsg, ProviderERROR)
			return object, err
		}
		err = WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
		return object, err
	}
	addDebug(action, response, request)
	v, err := jsonpath.Get("$", response)
	if err != nil {
		return object, WrapErrorf(err, FailedGetAttributeMsg, id, "$", response)
	}
	object = v.(map[string]interface{})
	if fmt.Sprint(object["VSwitchId"]) != id {
		return object, WrapErrorf(Error(GetNotFoundMessage("vswitch", id)), NotFoundMsg, ProviderERROR)
	}
	return object, nil
}

func (s *VpcService) VswitchStateRefreshFunc(id string, failStates []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		object, err := s.DescribeVswitch(id)
		if err != nil {
			if NotFoundError(err) {
				// Set this to nil as if we didn't find anything.
				return nil, "", nil
			}
			return nil, "", WrapError(err)
		}

		for _, failState := range failStates {
			if object["Status"].(string) == failState {
				return object, object["Status"].(string), WrapError(Error(FailedToReachTargetStatus, object["Status"].(string)))
			}
		}
		return object, object["Status"].(string), nil
	}
}

func (s *VpcService) SnatEntryStateRefreshFunc(id string, failStates []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		object, err := s.DescribeSnatEntry(id)
		if err != nil {
			if NotFoundError(err) {
				// Set this to nil as if we didn't find anything.
				return nil, "", nil
			}
			return nil, "", WrapError(err)
		}

		for _, failState := range failStates {
			if object["Status"].(string) == failState {
				return object, object["Status"].(string), WrapError(Error(FailedToReachTargetStatus, object["Status"].(string)))
			}
		}
		return object, object["Status"].(string), nil
	}
}

func (s *VpcService) ForwardEntryStateRefreshFunc(id string, failStates []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		object, err := s.DescribeForwardEntry(id)
		if err != nil {
			if NotFoundError(err) {
				// Set this to nil as if we didn't find anything.
				return nil, "", nil
			}
			return nil, "", WrapError(err)
		}

		for _, failState := range failStates {
			if object["Status"].(string) == failState {
				return object, object["Status"].(string), WrapError(Error(FailedToReachTargetStatus, object["Status"].(string)))
			}
		}
		return object, object["Status"].(string), nil
	}
}

func (s *VpcService) RouteTableStateRefreshFunc(id string, failStates []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		object, err := s.DescribeRouteTable(id)
		if err != nil {
			if NotFoundError(err) {
				// Set this to nil as if we didn't find anything.
				return nil, "", nil
			}
			return nil, "", WrapError(err)
		}

		for _, failState := range failStates {
			if object["Status"].(string) == failState {
				return object, object["Status"].(string), WrapError(Error(FailedToReachTargetStatus, object["Status"].(string)))
			}
		}
		return object, object["Status"].(string), nil
	}
}

func (s *VpcService) CommonBandwidthPackageStateRefreshFunc(id string, failStates []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		object, err := s.DescribeCommonBandwidthPackage(id)
		if err != nil {
			if NotFoundError(err) {
				// Set this to nil as if we didn't find anything.
				return nil, "", nil
			}
			return nil, "", WrapError(err)
		}

		for _, failState := range failStates {
			if object["Status"].(string) == failState {
				return object, object["Status"].(string), WrapError(Error(FailedToReachTargetStatus, object["Status"].(string)))
			}
		}
		return object, object["Status"].(string), nil
	}
}

func (s *VpcService) DescribeHavip(id string) (object map[string]interface{}, err error) {
	var response map[string]interface{}
	conn, err := s.client.NewVpcClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "DescribeHaVips"
	request := map[string]interface{}{
		"RegionId":   s.client.RegionId,
		"PageNumber": 1,
		"PageSize":   20,
	}
	for {
		runtime := util.RuntimeOptions{}
		runtime.SetAutoretry(true)
		response, err = conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2016-04-28"), StringPointer("AK"), nil, request, &runtime)
		if err != nil {
			if IsExpectedErrors(err, []string{"InvalidFilterKey.ValueNotSupported", "InvalidHaVipId.NotFound", "InvalidRegionId.NotFound"}) {
				err = WrapErrorf(Error(GetNotFoundMessage("Havip", id)), NotFoundMsg, ProviderERROR)
				return object, err
			}
			err = WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
			return object, err
		}
		addDebug(action, response, request)
		v, err := jsonpath.Get("$.HaVips.HaVip", response)
		if err != nil {
			return object, WrapErrorf(err, FailedGetAttributeMsg, id, "$.HaVips.HaVip", response)
		}
		result, _ := v.([]interface{})
		if len(result) < 1 {
			return object, WrapErrorf(Error(GetNotFoundMessage("VPC", id)), NotFoundWithResponse, response)
		}
		for _, v := range result {
			if v.(map[string]interface{})["HaVipId"].(string) == id {
				return v.(map[string]interface{}), nil
			}
		}
		if len(result) < request["PageSize"].(int) {
			break
		}
		request["PageNumber"] = request["PageNumber"].(int) + 1
	}
	return object, WrapErrorf(Error(GetNotFoundMessage("VPC", id)), NotFoundWithResponse, response)
}

func (s *VpcService) HavipStateRefreshFunc(id string, failStates []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		object, err := s.DescribeHavip(id)
		if err != nil {
			if NotFoundError(err) {
				// Set this to nil as if we didn't find anything.
				return nil, "", nil
			}
			return nil, "", WrapError(err)
		}

		for _, failState := range failStates {
			if object["Status"].(string) == failState {
				return object, object["Status"].(string), WrapError(Error(FailedToReachTargetStatus, object["Status"].(string)))
			}
		}
		return object, object["Status"].(string), nil
	}
}

func (s *VpcService) NatGatewayStateRefreshFunc(id string, failStates []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		object, err := s.DescribeNatGateway(id)
		if err != nil {
			if NotFoundError(err) {
				// Set this to nil as if we didn't find anything.
				return nil, "", nil
			}
			return nil, "", WrapError(err)
		}

		for _, failState := range failStates {
			if object["Status"].(string) == failState {
				return object, object["Status"].(string), WrapError(Error(FailedToReachTargetStatus, object["Status"].(string)))
			}
		}
		return object, object["Status"].(string), nil
	}
}
