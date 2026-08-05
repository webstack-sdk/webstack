package keeper_test

import (
	"testing"

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
	seedLicenseType(t, f, "webstack.node.gpu", creator)

	_, err := ms.CreateNodeType(f.Ctx, &types.MsgCreateNodeType{
		Creator:       creator,
		Id:            "webstack.gpu",
		LicenseTypeId: "webstack.node.gpu",
	})
	require.NoError(t, err)

	nt, err := f.Keeper.NodeTypes.Get(f.Ctx, "webstack.gpu")
	require.NoError(t, err)
	require.Equal(t, types.NodeType{
		Id:            "webstack.gpu",
		Creator:       creator,
		LicenseTypeId: "webstack.node.gpu",
	}, nt)

	// The inverse mapping is what makes "which node type does this license type
	// back" a single read, so its entry is part of the contract.
	bound, err := f.Keeper.NodeTypeByLicenseType.Get(f.Ctx, "webstack.node.gpu")
	require.NoError(t, err)
	require.Equal(t, "webstack.gpu", bound)
}

// TestCreateNodeTypeRequiresGrant: holding the license type is not enough. The
// creator of a license type still needs nodetype.create to define node types,
// so the right stays revocable on its own.
func TestCreateNodeTypeRequiresGrant(t *testing.T) {
	f, ms := setup(t)

	creator := sample.AccAddress()
	seedLicenseType(t, f, "webstack.node.gpu", creator)

	msg := &types.MsgCreateNodeType{
		Creator:       creator,
		Id:            "webstack.gpu",
		LicenseTypeId: "webstack.node.gpu",
	}

	_, err := ms.CreateNodeType(f.Ctx, msg)
	require.ErrorIs(t, err, types.ErrUnauthorized)
	require.ErrorContains(t, err, types.PermissionNodeTypeCreate)

	// Nothing was written on the rejected attempt.
	exists, err := f.Keeper.NodeTypes.Has(f.Ctx, "webstack.gpu")
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
	seedLicenseType(t, f, "webstack.node.gpu", owner)

	// The stranger is fully granted — only the creator match stops them.
	f.GrantNetwork(t, stranger, types.PermissionNodeTypeCreate)

	_, err := ms.CreateNodeType(f.Ctx, &types.MsgCreateNodeType{
		Creator:       stranger,
		Id:            "webstack.evil",
		LicenseTypeId: "webstack.node.gpu",
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
		LicenseTypeId: "webstack.node.gpu",
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
		Id:            "webstack.gpu",
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
	seedLicenseType(t, f, "webstack.node.gpu", creator)
	seedLicenseType(t, f, "webstack.node.other", creator)

	_, err := ms.CreateNodeType(f.Ctx, &types.MsgCreateNodeType{
		Creator:       creator,
		Id:            "webstack.gpu",
		LicenseTypeId: "webstack.node.gpu",
	})
	require.NoError(t, err)

	// Same id, different license type, same (authorized) creator.
	_, err = ms.CreateNodeType(f.Ctx, &types.MsgCreateNodeType{
		Creator:       creator,
		Id:            "webstack.gpu",
		LicenseTypeId: "webstack.node.other",
	})
	require.ErrorIs(t, err, types.ErrNodeTypeExists)

	nt, err := f.Keeper.NodeTypes.Get(f.Ctx, "webstack.gpu")
	require.NoError(t, err)
	require.Equal(t, "webstack.node.gpu", nt.LicenseTypeId, "the original binding must survive")
}

// TestCreateNodeTypeLicenseTypeAlreadyBound: the binding is one-to-one in both
// directions. A license type already backing a node type cannot back a second,
// even from its own creator — otherwise the two would collide in the inverse
// mapping and one would silently win.
func TestCreateNodeTypeLicenseTypeAlreadyBound(t *testing.T) {
	f, ms := setup(t)

	creator := sample.AccAddress()
	f.GrantNetwork(t, creator, types.PermissionNodeTypeCreate)
	seedLicenseType(t, f, "webstack.node.gpu", creator)

	_, err := ms.CreateNodeType(f.Ctx, &types.MsgCreateNodeType{
		Creator:       creator,
		Id:            "webstack.gpu",
		LicenseTypeId: "webstack.node.gpu",
	})
	require.NoError(t, err)

	_, err = ms.CreateNodeType(f.Ctx, &types.MsgCreateNodeType{
		Creator:       creator,
		Id:            "webstack.second",
		LicenseTypeId: "webstack.node.gpu",
	})
	require.ErrorIs(t, err, types.ErrLicenseTypeBound)
	require.ErrorContains(t, err, "webstack.gpu")

	// The rejected node type was not written, and the original binding stands.
	exists, err := f.Keeper.NodeTypes.Has(f.Ctx, "webstack.second")
	require.NoError(t, err)
	require.False(t, exists)

	bound, err := f.Keeper.NodeTypeByLicenseType.Get(f.Ctx, "webstack.node.gpu")
	require.NoError(t, err)
	require.Equal(t, "webstack.gpu", bound)
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
