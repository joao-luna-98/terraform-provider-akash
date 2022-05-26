package akash

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"os"
	"strconv"
	"terraform-provider-hashicups/akash/client"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDeployment() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDeploymentCreate,
		ReadContext:   resourceDeploymentRead,
		UpdateContext: resourceDeploymentUpdate,
		DeleteContext: resourceDeploymentDelete,
		Schema: map[string]*schema.Schema{
			"sdl": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"last_updated": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"deployment_state": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"deployment_dseq": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"deployment_owner": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"escrow_account_owner": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"escrow_account_balance_denom": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"escrow_account_balance_amount": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"escrow_account_state": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"provider_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceDeploymentCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	dseq, err := client.CreateDeployment(ctx, d.Get("sdl").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "Querying available bids")
	bids, err := client.GetBids(ctx, dseq, time.Second*60)
	if err != nil {
		return diag.FromErr(err)
	}
	if len(bids) == 0 {
		return diag.FromErr(errors.New("no bids on deployment"))
	}
	tflog.Info(ctx, fmt.Sprintf("Received %d bids in the deployment", len(bids)))

	// Select the provider
	provider := bids[0].Id.Provider
	tflog.Debug(ctx, fmt.Sprintf("Selected provider %s", provider))

	// Create a lease
	err = client.CreateLease(ctx, dseq, provider)
	if err != nil {
		return diag.FromErr(err)
	}

	// Send the manifest
	err = client.SendManifest(ctx, dseq, provider, "deployment.yaml")
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("deployment_dseq", dseq); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("deployment_owner", os.Getenv("AKASH_ACCOUNT_ADDRESS")); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("deployment_state", "active"); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("provider_address", provider); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return resourceDeploymentRead(ctx, d, m)
}

func resourceDeploymentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	//deploymentId := d.Id()

	// TODO: Get the deployment by Id
	deployment, err := client.GetDeployment(d.Get("deployment_dseq").(string), d.Get("deployment_owner").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("deployment_dseq", deployment["deployment_dseq"]); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("deployment_owner", deployment["deployment_owner"]); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("deployment_state", deployment["deployment_state"]); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("escrow_account_owner", deployment["escrow_account_owner"]); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("escrow_account_balance_denom", deployment["escrow_account_balance_denom"]); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("escrow_account_balance_amount", deployment["escrow_account_balance_amount"]); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("escrow_account_state", deployment["escrow_account_state"]); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceDeploymentUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	deploymentId := d.Id()
	fmt.Println(deploymentId)

	if d.HasChange("sdl") {

		// Update the deployment

		err := d.Set("last_updated", time.Now().Format(time.RFC850))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceDeploymentRead(ctx, d, m)
}

func resourceDeploymentDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	err := client.DeleteDeployment(d.Get("deployment_dseq").(string), d.Get("deployment_owner").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	// d.SetId("") is automatically called assuming delete returns no errors, but
	// it is added here for explicitness.
	d.SetId("")

	return diags
}
