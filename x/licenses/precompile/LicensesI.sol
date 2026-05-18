// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The default LicensesI precompile address. The chain may register the
///      precompile at a different address; consult chain documentation.
address constant LICENSES_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000001900;

/// @dev The LicensesI contract's instance.
LicensesI constant LICENSES_CONTRACT = LicensesI(LICENSES_PRECOMPILE_ADDRESS);

/// @dev LicensesParams represents the module params.
struct LicensesParams {
    address owner;
}

/// @dev LicenseType describes a class of issuable licenses.
struct LicenseType {
    string id;
    bool transferrable;
    uint256 maxSupply;
    uint256 issuedCount;
    uint256 activeCount;
    uint256 revokedCount;
}

/// @dev License is a single issued license.
struct License {
    uint64 id;
    string typeId;
    address holder;
    string startDate;
    string endDate;
    string status;
}

/// @dev AdminKeyGrant is one (permission, [licenseTypeId]) entry on an admin key.
struct AdminKeyGrant {
    string permission;
    string[] licenseTypes;
}

/// @dev AdminKey describes an address that has been granted admin permissions.
struct AdminKey {
    address adminAddress;
    AdminKeyGrant[] grants;
}

/// @dev AdminKeyPermission identifies a single (licenseTypeId, permission) pair
///      to revoke from an admin key.
struct AdminKeyPermission {
    string licenseTypeId;
    string permission;
}

/// @dev BatchIssueEntry is a single (holder, dates) tuple in a batch issuance.
struct BatchIssueEntry {
    address holder;
    string startDate;
    string endDate;
}

/// @author Webstack
/// @title Licenses Precompile Contract
/// @dev Exposes the x/licenses module to EVM smart contracts.
interface LicensesI {
    // ---------------------------------------------------------------------
    // Events
    // ---------------------------------------------------------------------

    /// @dev Emitted when a new license type is created.
    event LicenseTypeCreated(string indexed id, bool transferrable, uint256 maxSupply);

    /// @dev Emitted when a license type is updated.
    event LicenseTypeUpdated(string indexed id, bool transferrable, uint256 maxSupply);

    /// @dev Emitted when admin key permissions are granted (or merged) for an address.
    event AdminPermissionsGranted(address indexed admin);

    /// @dev Emitted when specific admin key permissions are revoked for an address.
    ///      The entire admin key entry is deleted if no grants remain.
    event AdminKeyPermissionsRevoked(address indexed admin);

    /// @dev Emitted when one or more licenses of a single type are issued to a holder.
    event LicenseIssued(
        address indexed issuer,
        address indexed holder,
        string licenseTypeId,
        uint64 count
    );

    /// @dev Emitted when one or more licenses are revoked from a holder.
    event LicenseRevoked(
        address indexed revoker,
        address indexed holder,
        string licenseTypeId,
        uint64 count
    );

    /// @dev Emitted when a single license is transferred between holders.
    event LicenseTransferred(
        address indexed from,
        address indexed to,
        string licenseTypeId,
        uint64 id
    );

    /// @dev Emitted when a batch of licenses are issued.
    event LicenseBatchIssued(
        address indexed issuer,
        string licenseTypeId,
        uint64 count
    );

    // ---------------------------------------------------------------------
    // Transactions
    // ---------------------------------------------------------------------

    /// @dev Create a new license type. Caller must be the module owner.
    function createLicenseType(
        string calldata id,
        bool transferrable,
        uint256 maxSupply
    ) external returns (bool success);

    /// @dev Update an existing license type. Caller must be the module owner.
    function updateLicenseType(
        string calldata id,
        bool transferrable,
        uint256 maxSupply
    ) external returns (bool success);

    /// @dev Grant admin key permissions for an address. The supplied grants are
    ///      MERGED with any existing grants; (permission, licenseType) pairs that
    ///      already exist are deduped. Caller must be the module owner.
    function grantAdminPermissions(
        address admin,
        AdminKeyGrant[] calldata grants
    ) external returns (bool success);

    /// @dev Revoke specific (licenseTypeId, permission) pairs from an admin key.
    ///      Pairs that are not currently granted are silently ignored. A grant
    ///      whose license types become empty is dropped; if no grants remain
    ///      the admin key entry itself is deleted. Caller must be the module owner.
    function revokeAdminKeyPermissions(
        address admin,
        AdminKeyPermission[] calldata permissions
    ) external returns (bool success);

    /// @dev Issue one or more licenses of the given type to a holder.
    ///      Caller must hold the `issue` permission for that license type.
    ///      Dates are formatted as YYYY-MM-DD.
    function issueLicense(
        string calldata licenseTypeId,
        address holder,
        string calldata startDate,
        string calldata endDate,
        uint64 count
    ) external returns (uint64[] memory ids);

    /// @dev Revoke `count` active licenses (most recently issued first) of the given
    ///      type from `holder`. Caller must hold the `revoke` permission.
    function revokeLicense(
        string calldata licenseTypeId,
        address holder,
        uint64 count
    ) external returns (uint64[] memory ids);

    /// @dev Transfer a license to a new holder. Caller must be the current holder.
    function transferLicense(
        string calldata licenseTypeId,
        uint64 id,
        address recipient
    ) external returns (bool success);

    /// @dev Issue a batch of licenses of a single type with per-entry holders and dates.
    function batchIssueLicense(
        string calldata licenseTypeId,
        BatchIssueEntry[] calldata entries
    ) external returns (uint64[] memory ids);

    // ---------------------------------------------------------------------
    // Queries
    // ---------------------------------------------------------------------

    /// @dev Returns module params.
    function params() external view returns (LicensesParams memory);

    /// @dev Returns the set of valid grant permission strings.
    function permissions() external view returns (string[] memory);

    /// @dev Returns a single license type by id. Reverts if not found.
    function licenseType(string calldata id) external view returns (LicenseType memory);

    /// @dev Returns all license types.
    function licenseTypes() external view returns (LicenseType[] memory);

    /// @dev Returns a single license by type+id. Reverts if not found.
    function license(string calldata typeId, uint64 id) external view returns (License memory);

    /// @dev Returns all licenses of the given type.
    function licensesByType(string calldata typeId) external view returns (License[] memory);

    /// @dev Returns all licenses held by `holder`.
    function licensesByHolder(address holder) external view returns (License[] memory);

    /// @dev Returns all licenses of a given type held by `holder`.
    function licensesByHolderAndType(
        address holder,
        string calldata typeId
    ) external view returns (License[] memory);

    /// @dev Returns the admin key entry for an address. Reverts if not found.
    function adminKey(address admin) external view returns (AdminKey memory);

    /// @dev Returns all admin keys.
    function adminKeys() external view returns (AdminKey[] memory);

    /// @dev Returns admin keys that have `permission` over `licenseTypeId`.
    ///      An empty `permission` matches any permission.
    function adminKeysByLicenseType(
        string calldata licenseTypeId,
        string calldata permission
    ) external view returns (AdminKey[] memory);
}
