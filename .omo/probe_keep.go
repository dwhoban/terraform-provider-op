package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/1password/onepassword-sdk-go"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	opts := []onepassword.ClientOption{
		onepassword.WithIntegrationInfo("terraform-provider-op-probe", "0.0.0"),
		onepassword.WithDesktopAppIntegration(os.Getenv("OP_DESKTOP_ACCOUNT_NAME")),
	}
	client, err := onepassword.NewClient(ctx, opts...)
	if err != nil {
		fmt.Printf("client error: %v\n", err)
		os.Exit(1)
	}
	vaultID := os.Getenv("OP_PROBE_VAULT_ID")
	item, err := client.Items().Create(ctx, onepassword.ItemCreateParams{
		Title:    fmt.Sprintf("SDK SSH probe keep %d", time.Now().Unix()),
		Category: onepassword.ItemCategorySSHKey,
		VaultID:  vaultID,
	})
	if err != nil {
		fmt.Printf("CREATE: error: %v\n", err)
		os.Exit(0)
	}
	fmt.Printf("CREATE: ok item=%s fields=%d\n", item.ID, len(item.Fields))
	got, err := client.Items().Get(ctx, vaultID, item.ID)
	if err != nil {
		fmt.Printf("GET: error: %v\n", err)
		return
	}
	fmt.Printf("GET: fields=%d\n", len(got.Fields))
	for _, f := range got.Fields {
		fmt.Printf("  %q type=%s valueLen=%d", f.Title, f.FieldType, len(f.Value))
		if f.Details != nil && f.Details.SSHKey() != nil {
			fmt.Printf(" sshKey{type=%s fp=%s}", f.Details.SSHKey().KeyType, f.Details.SSHKey().Fingerprint)
		}
		fmt.Println()
	}
}
