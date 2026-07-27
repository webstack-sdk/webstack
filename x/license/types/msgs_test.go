package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/webstack-sdk/webstack/testutil/sample"
	"github.com/webstack-sdk/webstack/x/license/types"
)

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
