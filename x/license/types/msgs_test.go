package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/x/tx/signing"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	licensev1 "github.com/webstack-sdk/webstack/api/license/v1"
	"github.com/webstack-sdk/webstack/testutil/sample"
	"github.com/webstack-sdk/webstack/x/license/types"
)

// TestMsgSignerAnnotations resolves signers the way the SDK does at tx
// handling time, from the cosmos.msg.v1.signer proto annotation rather than
// the Go struct. A rename that leaves the annotation pointing at a field that
// no longer exists still compiles, so this is the check that catches it.
func TestMsgSignerAnnotations(t *testing.T) {
	prefix := sdk.GetConfig().GetBech32AccountAddrPrefix()
	registry, err := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec:          address.NewBech32Codec(prefix),
			ValidatorAddressCodec: address.NewBech32Codec(prefix + "valoper"),
		},
	})
	require.NoError(t, err)
	types.RegisterInterfaces(registry)

	addr := sample.AccAddress()
	want, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)

	// CreateLicenseType signs as "creator": the holder of type.create, which
	// is not necessarily the namespace owner.
	signers, err := registry.SigningContext().GetSigners(&licensev1.MsgCreateLicenseType{Creator: addr})
	require.NoError(t, err)
	require.Equal(t, [][]byte{want}, signers)

	// UpdateLicenseType still signs as "owner" — it remains owner-gated.
	signers, err = registry.SigningContext().GetSigners(&licensev1.MsgUpdateLicenseType{Owner: addr})
	require.NoError(t, err)
	require.Equal(t, [][]byte{want}, signers)
}

func TestMsgUpdateLicenseTypeValidateBasic(t *testing.T) {
	owner := sample.AccAddress()

	require.NoError(t, (&types.MsgUpdateLicenseType{Owner: owner, Id: "lt1"}).ValidateBasic())
	require.NoError(t, (&types.MsgUpdateLicenseType{Owner: owner, Id: "lt1", Transferrable: true}).ValidateBasic())

	require.ErrorContains(t,
		(&types.MsgUpdateLicenseType{Owner: "x", Id: "lt1"}).ValidateBasic(),
		"invalid owner address")
	require.ErrorContains(t,
		(&types.MsgUpdateLicenseType{Owner: owner}).ValidateBasic(),
		"license type id cannot be empty")
}
