package keeper_test

import (
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/webstack-sdk/webstack/testutil/keeper"
	"github.com/webstack-sdk/webstack/testutil/sample"
	licensetypes "github.com/webstack-sdk/webstack/x/license/types"
	"github.com/webstack-sdk/webstack/x/network/types"
)

// ---------------------------------------------------------------------------
// CreateNodeType
// ---------------------------------------------------------------------------

// seedLicenseType writes a license type owned by creator straight into license
// state. The license msg path has its own tests; what matters here is only
// which address is recorded as the creator.
func seedLicenseType(t testing.TB, f *keepertest.NetworkFixture, id, creator string) {
	t.Helper()
	require.NoError(t, f.LicenseKeeper.LicenseTypes.Set(f.Ctx, id, licensetypes.LicenseType{
		Id:           id,
		Creator:      creator,
		MaxSupply:    math.ZeroInt(),
		IssuedCount:  math.ZeroInt(),
		ActiveCount:  math.ZeroInt(),
		RevokedCount: math.ZeroInt(),
	}))
}

// TestCreateNodeType covers the happy path: both gates satisfied writes the
// record and its by-license index entry.
func TestCreateNodeType(t *testing.T) {
	f, ms := setup(t)

	creator := sample.AccAddress()
	f.GrantNetwork(t, creator, types.PermissionNodeTypeCreate)
	seedLicenseType(t, f, "webstack.node", creator)

	_, err := ms.CreateNodeType(f.Ctx, &types.MsgCreateNodeType{
		Creator:       creator,
		Id:            "webstack.trust",
		LicenseTypeId: "webstack.node",
	})
	require.NoError(t, err)

	nt, err := f.Keeper.NodeTypes.Get(f.Ctx, "webstack.trust")
	require.NoError(t, err)
	require.Equal(t, types.NodeType{
		Id:            "webstack.trust",
		Creator:       creator,
		LicenseTypeId: "webstack.node",
	}, nt)

	// The reverse index is what makes "node types for this license" a prefix
	// walk rather than a full scan, so its entry is part of the contract.
	indexed, err := f.Keeper.NodeTypesByLicense.Has(f.Ctx, collections.Join("webstack.node", "webstack.trust"))
	require.NoError(t, err)
	require.True(t, indexed)
}

// TestCreateNodeTypeRequiresGrant: holding the license type is not enough. The
// creator of a license type still needs nodetype.create to define node types,
// so the right stays revocable on its own.
func TestCreateNodeTypeRequiresGrant(t *testing.T) {
	f, ms := setup(t)

	creator := sample.AccAddress()
	seedLicenseType(t, f, "webstack.node", creator)

	msg := &types.MsgCreateNodeType{
		Creator:       creator,
		Id:            "webstack.trust",
		LicenseTypeId: "webstack.node",
	}

	_, err := ms.CreateNodeType(f.Ctx, msg)
	require.ErrorIs(t, err, types.ErrUnauthorized)
	require.ErrorContains(t, err, types.PermissionNodeTypeCreate)

	// Nothing was written on the rejected attempt.
	exists, err := f.Keeper.NodeTypes.Has(f.Ctx, "webstack.trust")
	require.NoError(t, err)
	require.False(t, exists)

	f.GrantNetwork(t, creator, types.PermissionNodeTypeCreate)
	_, err = ms.CreateNodeType(f.Ctx, msg)
	require.NoError(t, err)
}

// TestCreateNodeTypeRequiresLicenseTypeCreator is the core new rule: the grant
// alone does not let one address attach node types to another's license type.
func TestCreateNodeTypeRequiresLicenseTypeCreator(t *testing.T) {
	f, ms := setup(t)

	owner := sample.AccAddress()
	stranger := sample.AccAddress()
	seedLicenseType(t, f, "webstack.node", owner)

	// The stranger is fully granted — only the creator match stops them.
	f.GrantNetwork(t, stranger, types.PermissionNodeTypeCreate)

	_, err := ms.CreateNodeType(f.Ctx, &types.MsgCreateNodeType{
		Creator:       stranger,
		Id:            "webstack.evil",
		LicenseTypeId: "webstack.node",
	})
	require.ErrorIs(t, err, types.ErrUnauthorized)
	require.ErrorContains(t, err, "did not create license type")

	exists, err := f.Keeper.NodeTypes.Has(f.Ctx, "webstack.evil")
	require.NoError(t, err)
	require.False(t, exists)

	// The same message from the license type's creator is accepted, so the
	// rejection above is about identity and nothing else.
	f.GrantNetwork(t, owner, types.PermissionNodeTypeCreate)
	_, err = ms.CreateNodeType(f.Ctx, &types.MsgCreateNodeType{
		Creator:       owner,
		Id:            "webstack.evil",
		LicenseTypeId: "webstack.node",
	})
	require.NoError(t, err)
}

// TestCreateNodeTypeUnknownLicenseType: a binding to a license type that does
// not exist is a distinct failure from an authorization one.
func TestCreateNodeTypeUnknownLicenseType(t *testing.T) {
	f, ms := setup(t)

	creator := sample.AccAddress()
	f.GrantNetwork(t, creator, types.PermissionNodeTypeCreate)

	_, err := ms.CreateNodeType(f.Ctx, &types.MsgCreateNodeType{
		Creator:       creator,
		Id:            "webstack.trust",
		LicenseTypeId: "does.not.exist",
	})
	require.ErrorIs(t, err, types.ErrLicenseTypeNotFound)
}

// TestCreateNodeTypeDuplicate: ids are single-use. A second registration must
// not silently repoint an existing type's license binding, because nodes
// already carry that type.
func TestCreateNodeTypeDuplicate(t *testing.T) {
	f, ms := setup(t)

	creator := sample.AccAddress()
	f.GrantNetwork(t, creator, types.PermissionNodeTypeCreate)
	seedLicenseType(t, f, "webstack.node", creator)
	seedLicenseType(t, f, "webstack.other", creator)

	_, err := ms.CreateNodeType(f.Ctx, &types.MsgCreateNodeType{
		Creator:       creator,
		Id:            "webstack.trust",
		LicenseTypeId: "webstack.node",
	})
	require.NoError(t, err)

	// Same id, different license type, same (authorized) creator.
	_, err = ms.CreateNodeType(f.Ctx, &types.MsgCreateNodeType{
		Creator:       creator,
		Id:            "webstack.trust",
		LicenseTypeId: "webstack.other",
	})
	require.ErrorIs(t, err, types.ErrNodeTypeExists)

	nt, err := f.Keeper.NodeTypes.Get(f.Ctx, "webstack.trust")
	require.NoError(t, err)
	require.Equal(t, "webstack.node", nt.LicenseTypeId, "the original binding must survive")
}

// TestCreateNodeTypeValidateBasic pins the stateless checks.
func TestCreateNodeTypeValidateBasic(t *testing.T) {
	creator := sample.AccAddress()

	require.NoError(t, (&types.MsgCreateNodeType{
		Creator: creator, Id: "t", LicenseTypeId: "l",
	}).ValidateBasic())

	err := (&types.MsgCreateNodeType{Creator: creator, LicenseTypeId: "l"}).ValidateBasic()
	require.ErrorIs(t, err, types.ErrInvalidNodeType)

	err = (&types.MsgCreateNodeType{Creator: creator, Id: "t"}).ValidateBasic()
	require.ErrorIs(t, err, types.ErrLicenseTypeNotFound)
}
