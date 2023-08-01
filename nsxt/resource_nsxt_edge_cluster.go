/* Copyright © 2023 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: MPL-2.0 */

package nsxt

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt-mp/nsx"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt-mp/nsx/model"
)

func resourceNsxtEdgeCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceNsxtEdgeClusterCreate,
		Read:   resourceNsxtEdgeClusterRead,
		Update: resourceNsxtEdgeClusterUpdate,
		Delete: resourceNsxtEdgeClusterDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"revision":     getRevisionSchema(),
			"description":  getDescriptionSchema(),
			"display_name": getDisplayNameSchema(),
			"tag":          getTagsSchema(),
			"edge_ha_profile_id": {
				Type:        schema.TypeString,
				Description: "Edge high availability cluster profile Id",
				Optional:    true,
				Computed:    true,
			},
			"member_node_type": {
				Type:        schema.TypeString,
				Description: "Node type of the cluster members",
				Computed:    true,
			},
			"member": {
				Type:        schema.TypeList,
				Description: "Edge cluster members",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"description": {
							Type:        schema.TypeString,
							Description: "Description of this resource",
							Optional:    true,
						},
						"display_name": {
							Type:        schema.TypeString,
							Description: "The display name of this resource. Defaults to ID if not set",
							Optional:    true,
							Computed:    true,
						},
						"member_index": {
							Type:        schema.TypeInt,
							Description: "System generated index for cluster member",
							Computed:    true,
						},
						"transport_node_id": {
							Type:        schema.TypeString,
							Description: "UUID of edge transport node",
							Required:    true,
						},
					},
				},
			},
			"node_rtep_ips": {
				Type:        schema.TypeList,
				Description: "Remote tunnel endpoint ip address",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"member_index": {
							Type:        schema.TypeInt,
							Description: "System generated index for cluster member",
							Computed:    true,
						},
						"rtep_ips": {
							Type:        schema.TypeList,
							Description: "Remote tunnel endpoint ip address",
							Computed:    true,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validateSingleIP(),
							},
						},
						"transport_node_id": {
							Type:        schema.TypeString,
							Description: "UUID of edge transport node",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func resourceNsxtEdgeClusterCreate(d *schema.ResourceData, m interface{}) error {
	connector := getPolicyConnector(m)
	client := nsx.NewEdgeClustersClient(connector)

	description := d.Get("description").(string)
	displayName := d.Get("display_name").(string)
	tags := getMPTagsFromSchema(d)
	clusterProfileBindings := getClusterProfileBindingsFromSchema(d)

	members := getEdgeClusterMembersFromSchema(d)
	obj := model.EdgeCluster{
		Description:            &description,
		DisplayName:            &displayName,
		Tags:                   tags,
		ClusterProfileBindings: clusterProfileBindings,
		Members:                members,
	}

	obj, err := client.Create(obj)
	if err != nil {
		id := ""
		if obj.Id != nil {
			id = *obj.Id
		}
		return handleCreateError("Edge Cluster", id, err)
	}

	log.Printf("[INFO] Creating Edge Cluster with ID %s", *obj.Id)

	d.SetId(*obj.Id)
	return resourceNsxtEdgeClusterRead(d, m)
}

func getClusterProfileBindingsFromSchema(d *schema.ResourceData) []model.ClusterProfileTypeIdEntry {
	resourceType := model.ClusterProfileTypeIdEntry_RESOURCE_TYPE_EDGEHIGHAVAILABILITYPROFILE
	clusterProfileBinding := d.Get("edge_ha_profile_id").(string)
	var clusterProfileBindings []model.ClusterProfileTypeIdEntry
	if clusterProfileBinding != "" {
		clusterProfileBindings = []model.ClusterProfileTypeIdEntry{
			{
				ProfileId:    &clusterProfileBinding,
				ResourceType: &resourceType,
			},
		}
	}
	return clusterProfileBindings
}

func getEdgeClusterMembersFromSchema(d *schema.ResourceData) []model.EdgeClusterMember {
	memberList := d.Get("member").([]interface{})
	var members []model.EdgeClusterMember
	for _, member := range memberList {
		data := member.(map[string]interface{})
		description := data["description"].(string)
		displayName := data["display_name"].(string)
		memberIndex := data["member_index"].(int64)
		transportNodeID := data["transport_node_id"].(string)
		elem := model.EdgeClusterMember{
			Description:     &description,
			DisplayName:     &displayName,
			MemberIndex:     &memberIndex,
			TransportNodeId: &transportNodeID,
		}
		members = append(members, elem)
	}
	return members
}

func resourceNsxtEdgeClusterRead(d *schema.ResourceData, m interface{}) error {
	connector := getPolicyConnector(m)
	id := d.Id()
	if id == "" {
		return fmt.Errorf("error obtaining logical object id")
	}

	client := nsx.NewEdgeClustersClient(connector)
	obj, err := client.Get(id)
	if err != nil {
		return fmt.Errorf("error during Edge Cluster read: %v", err)
	}

	d.Set("revision", obj.Revision)
	d.Set("description", obj.Description)
	d.Set("display_name", obj.DisplayName)
	setMPTagsInSchema(d, obj.Tags)

	setClusterProfileBindingsInSchema(d, obj)

	d.Set("member_node_type", obj.MemberNodeType)
	setMemberListInSchema(d, obj.Members)
	setNodeRtepIPsInSchema(d, obj.NodeRtepIps)
	return nil
}

func setClusterProfileBindingsInSchema(d *schema.ResourceData, obj model.EdgeCluster) {
	for _, cpb := range obj.ClusterProfileBindings {
		if *cpb.ResourceType == model.ClusterProfileTypeIdEntry_RESOURCE_TYPE_EDGEHIGHAVAILABILITYPROFILE {
			d.Set("edge_ha_profile_id", *cpb.ProfileId)
			// Model contains a single profile id
			return
		}
		log.Printf("Unsupported resource %s", *cpb.ResourceType)
	}
}

func setNodeRtepIPsInSchema(d *schema.ResourceData, nodeRtepIPs []model.NodeRtepIpsConfig) error {
	var expressionList []map[string]interface{}
	for _, rtepIP := range nodeRtepIPs {
		elem := make(map[string]interface{})
		elem["member_index"] = rtepIP.MemberIndex
		elem["rtep_ips"] = rtepIP.RtepIps
		elem["transport_node_id"] = rtepIP.TransportNodeId
		expressionList = append(expressionList, elem)
	}
	return d.Set("node_rtep_ips", expressionList)
}

func setMemberListInSchema(d *schema.ResourceData, members []model.EdgeClusterMember) error {
	var expresionList []map[string]interface{}
	for _, member := range members {
		elem := make(map[string]interface{})
		elem["description"] = member.Description
		elem["display_name"] = member.DisplayName
		elem["member_index"] = member.MemberIndex
		elem["transport_node_id"] = member.TransportNodeId
		expresionList = append(expresionList, elem)
	}
	return d.Set("member", expresionList)
}

func resourceNsxtEdgeClusterUpdate(d *schema.ResourceData, m interface{}) error {
	connector := getPolicyConnector(m)
	id := d.Id()
	if id == "" {
		return fmt.Errorf("error obtaining logical object id")
	}

	client := nsx.NewEdgeClustersClient(connector)

	revision := int64(d.Get("revision").(int))
	description := d.Get("description").(string)
	displayName := d.Get("display_name").(string)
	tags := getMPTagsFromSchema(d)
	members := getEdgeClusterMembersFromSchema(d)
	clusterProfileBindings := getClusterProfileBindingsFromSchema(d)
	obj := model.EdgeCluster{
		Revision:               &revision,
		Description:            &description,
		DisplayName:            &displayName,
		Tags:                   tags,
		ClusterProfileBindings: clusterProfileBindings,
		Members:                members,
	}

	_, err := client.Update(id, obj)
	if err != nil {
		return fmt.Errorf("error during Edge Cluster %s update: %v", id, err)
	}

	return resourceNsxtEdgeClusterRead(d, m)
}

func resourceNsxtEdgeClusterDelete(d *schema.ResourceData, m interface{}) error {
	connector := getPolicyConnector(m)

	id := d.Id()
	if id == "" {
		return fmt.Errorf("error obtaining logical object id")
	}

	client := nsx.NewEdgeClustersClient(connector)

	err := client.Delete(id)
	if err != nil {
		return fmt.Errorf("error during Edge Cluster delete: %v", err)
	}
	return nil
}
