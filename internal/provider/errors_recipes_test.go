// Copyright (c) 2026 Dan Hoban
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"errors"
	"testing"

	"github.com/1password/onepassword-sdk-go"
	tfsdk "github.com/hashicorp/terraform-plugin-framework/types"
)

func TestIsNotFound(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("item not found"), true},
		{errors.New("Item is not in an active state"), true},
		{errors.New("permission denied"), false},
		{errors.New("rate limit exceeded"), false},
	}
	for _, c := range cases {
		if got := isNotFound(c.err); got != c.want {
			t.Errorf("isNotFound(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}

func TestPasswordRecipeFromConfig(t *testing.T) {
	str := func(s string) tfsdk.String { return tfsdk.StringValue(s) }

	random, err := passwordRecipeFromConfig(&generatePasswordActionModel{
		Mode:           str("random"),
		Length:         tfsdk.Int64Value(20),
		IncludeDigits:  tfsdk.BoolValue(false),
		IncludeSymbols: tfsdk.BoolValue(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	inner := random.Random()
	if inner == nil || inner.Length != 20 || inner.IncludeDigits || !inner.IncludeSymbols {
		t.Fatalf("random recipe mismatch: %+v", inner)
	}

	pin, err := passwordRecipeFromConfig(&generatePasswordActionModel{Mode: str("pin")})
	if err != nil {
		t.Fatal(err)
	}
	if pinInner := pin.Pin(); pinInner == nil || pinInner.Length != 6 {
		t.Fatalf("pin default length should be 6, got %+v", pinInner)
	}

	memorable, err := passwordRecipeFromConfig(&generatePasswordActionModel{Mode: str("memorable")})
	if err != nil {
		t.Fatal(err)
	}
	memInner := memorable.Memorable()
	if memInner == nil || memInner.WordCount != 4 || memInner.WordListType != onepassword.WordListTypeFullWords {
		t.Fatalf("memorable defaults mismatch: %+v", memInner)
	}

	if _, err := passwordRecipeFromConfig(&generatePasswordActionModel{Mode: str("dsa")}); err == nil {
		t.Fatal("expected error for unknown mode")
	}
	if _, err := passwordRecipeFromConfig(&generatePasswordActionModel{Mode: str("random"), Length: tfsdk.Int64Value(100)}); err == nil {
		t.Fatal("expected error for out-of-range length")
	}
}
