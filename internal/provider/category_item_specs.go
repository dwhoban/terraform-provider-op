// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"

	"github.com/1password/onepassword-sdk-go"
)

// categoryFieldSpec describes one typed top-level attribute of a category
// item resource and the canonical 1Password field it maps to.
type categoryFieldSpec struct {
	Attr      string                    // HCL attribute name, e.g. "valid_from"
	FieldID   string                    // canonical 1Password field ID, e.g. "validFrom"
	Title     string                    // field label shown in 1Password, e.g. "valid from"
	Type      onepassword.ItemFieldType // SDK type, e.g. ItemFieldTypeMenu
	Sensitive bool
	Address   bool   // true: nested address object instead of string value (Address type)
	SectionID string // template section the field belongs to; "" = main section
}

// sshDetailSpec describes a computed attribute derived from the SSH key
// details of a source field (SshKey category only).
type sshDetailSpec struct {
	Attr      string // computed HCL attr
	FieldID   string // source field whose Details to read
	DetailKey string // "public_key" | "fingerprint" | "key_type"
}

// itemSectionSpec is a default section defined by the category template.
type itemSectionSpec struct {
	ID    string // canonical section ID, e.g. "contactInfo"
	Title string // section label, e.g. "Contact Information"
}

// categorySpec drives one op_item_<category> resource.
type categorySpec struct {
	TypeName        string // suffix appended to "op", e.g. "item_ssh_key"
	Category        onepassword.ItemCategory
	Description     string
	Fields          []categoryFieldSpec // typed top-level attributes
	SSHDetails      []sshDetailSpec     // SshKey category only
	DefaultSections []itemSectionSpec   // template sections; engine sends those referenced by written fields
	Websites        bool                // include `websites` attribute: Login, Password, ApiCredentials
	GenerateSSHKey  bool                // include `generate_ssh_key`: SshKey
}

// Category item specs mirror the default field set 1Password itself creates
// per category: canonical field IDs, titles, types, and template sections
// from `op item template get <category>` dumps (op 2.42.0-beta.01,
// 2026-11-12). Every template also carries a notesPlain field,
// which is never typed: the shared `notes` attribute maps to it.
var categoryItemSpecs = []categorySpec{
	{
		TypeName:    "item_login",
		Category:    onepassword.ItemCategoryLogin,
		Description: "Manages a 1Password Login item.",
		Fields: []categoryFieldSpec{
			{Attr: "username", FieldID: "username", Title: "username", Type: onepassword.ItemFieldTypeText},
			{Attr: "password", FieldID: "password", Title: "password", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true},
		},
		Websites: true,
	},
	{
		TypeName:    "item_credit_card",
		Category:    onepassword.ItemCategoryCreditCard,
		Description: "Manages a 1Password CreditCard item.",
		DefaultSections: []itemSectionSpec{
			{ID: "details", Title: "Additional Details"},
			{ID: "contactInfo", Title: "Contact Information"},
		},
		Fields: []categoryFieldSpec{
			{Attr: "cardholder", FieldID: "cardholder", Title: "cardholder name", Type: onepassword.ItemFieldTypeText},
			{Attr: "card_type", FieldID: "type", Title: "type", Type: onepassword.ItemFieldTypeCreditCardType},
			{Attr: "card_number", FieldID: "ccnum", Title: "number", Type: onepassword.ItemFieldTypeCreditCardNumber, Sensitive: true},
			{Attr: "cvv", FieldID: "cvv", Title: "verification number", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true},
			{Attr: "expiry", FieldID: "expiry", Title: "expiry date", Type: onepassword.ItemFieldTypeMonthYear},
			{Attr: "valid_from", FieldID: "validFrom", Title: "valid from", Type: onepassword.ItemFieldTypeMonthYear},
			{Attr: "issuing_bank", FieldID: "bank", Title: "issuing bank", Type: onepassword.ItemFieldTypeText, SectionID: "contactInfo"},
			{Attr: "phone_local", FieldID: "phoneLocal", Title: "phone (local)", Type: onepassword.ItemFieldTypePhone, SectionID: "contactInfo"},
			{Attr: "phone_toll_free", FieldID: "phoneTollFree", Title: "phone (toll free)", Type: onepassword.ItemFieldTypePhone, SectionID: "contactInfo"},
			{Attr: "phone_international", FieldID: "phoneIntl", Title: "phone (intl)", Type: onepassword.ItemFieldTypePhone, SectionID: "contactInfo"},
			{Attr: "website", FieldID: "website", Title: "website", Type: onepassword.ItemFieldTypeURL, SectionID: "contactInfo"},
			{Attr: "pin", FieldID: "pin", Title: "PIN", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true, SectionID: "details"},
			{Attr: "credit_limit", FieldID: "creditLimit", Title: "credit limit", Type: onepassword.ItemFieldTypeText, SectionID: "details"},
			{Attr: "cash_withdrawal_limit", FieldID: "cashLimit", Title: "cash withdrawal limit", Type: onepassword.ItemFieldTypeText, SectionID: "details"},
			{Attr: "interest_rate", FieldID: "interest", Title: "interest rate", Type: onepassword.ItemFieldTypeText, SectionID: "details"},
			{Attr: "issue_number", FieldID: "issuenumber", Title: "issue number", Type: onepassword.ItemFieldTypeText, SectionID: "details"},
		},
	},
	{
		TypeName:    "item_secure_note",
		Category:    onepassword.ItemCategorySecureNote,
		Description: "Manages a 1Password SecureNote item.",
	},
	{
		TypeName:    "item_identity",
		Category:    onepassword.ItemCategoryIdentity,
		Description: "Manages a 1Password Identity item.",
		DefaultSections: []itemSectionSpec{
			{ID: "name", Title: "Identification"},
			{ID: "address", Title: "Address"},
			{ID: "internet", Title: "Internet Details"},
		},
		Fields: []categoryFieldSpec{
			{Attr: "first_name", FieldID: "firstname", Title: "first name", Type: onepassword.ItemFieldTypeText, SectionID: "name"},
			{Attr: "initial", FieldID: "initial", Title: "initial", Type: onepassword.ItemFieldTypeText, SectionID: "name"},
			{Attr: "last_name", FieldID: "lastname", Title: "last name", Type: onepassword.ItemFieldTypeText, SectionID: "name"},
			{Attr: "gender", FieldID: "gender", Title: "gender", Type: onepassword.ItemFieldTypeText, SectionID: "name"},
			{Attr: "birth_date", FieldID: "birthdate", Title: "birth date", Type: onepassword.ItemFieldTypeDate, SectionID: "name"},
			{Attr: "occupation", FieldID: "occupation", Title: "occupation", Type: onepassword.ItemFieldTypeText, SectionID: "name"},
			{Attr: "company", FieldID: "company", Title: "company", Type: onepassword.ItemFieldTypeText, SectionID: "name"},
			{Attr: "department", FieldID: "department", Title: "department", Type: onepassword.ItemFieldTypeText, SectionID: "name"},
			{Attr: "job_title", FieldID: "jobtitle", Title: "job title", Type: onepassword.ItemFieldTypeText, SectionID: "name"},
			{Attr: "address", FieldID: "address", Title: "address", Type: onepassword.ItemFieldTypeAddress, Address: true, SectionID: "address"},
			{Attr: "default_phone", FieldID: "defphone", Title: "default phone", Type: onepassword.ItemFieldTypePhone, SectionID: "address"},
			{Attr: "home_phone", FieldID: "homephone", Title: "home", Type: onepassword.ItemFieldTypePhone, SectionID: "address"},
			{Attr: "cell_phone", FieldID: "cellphone", Title: "cell", Type: onepassword.ItemFieldTypePhone, SectionID: "address"},
			{Attr: "business_phone", FieldID: "busphone", Title: "business", Type: onepassword.ItemFieldTypePhone, SectionID: "address"},
			{Attr: "username", FieldID: "username", Title: "username", Type: onepassword.ItemFieldTypeText, SectionID: "internet"},
			{Attr: "reminder_question", FieldID: "reminderq", Title: "reminder question", Type: onepassword.ItemFieldTypeText, SectionID: "internet"},
			{Attr: "reminder_answer", FieldID: "remindera", Title: "reminder answer", Type: onepassword.ItemFieldTypeText, SectionID: "internet"},
			{Attr: "email", FieldID: "email", Title: "email", Type: onepassword.ItemFieldTypeEmail, SectionID: "internet"},
			{Attr: "website", FieldID: "website", Title: "website", Type: onepassword.ItemFieldTypeText, SectionID: "internet"},
			{Attr: "icq", FieldID: "icq", Title: "ICQ", Type: onepassword.ItemFieldTypeText, SectionID: "internet"},
			{Attr: "skype", FieldID: "skype", Title: "skype", Type: onepassword.ItemFieldTypeText, SectionID: "internet"},
			{Attr: "aim", FieldID: "aim", Title: "AOL/AIM", Type: onepassword.ItemFieldTypeText, SectionID: "internet"},
			{Attr: "yahoo", FieldID: "yahoo", Title: "Yahoo", Type: onepassword.ItemFieldTypeText, SectionID: "internet"},
			{Attr: "msn", FieldID: "msn", Title: "MSN", Type: onepassword.ItemFieldTypeText, SectionID: "internet"},
			{Attr: "forum_signature", FieldID: "forumsig", Title: "forum signature", Type: onepassword.ItemFieldTypeText, SectionID: "internet"},
		},
	},
	{
		TypeName:    "item_password",
		Category:    onepassword.ItemCategoryPassword,
		Description: "Manages a 1Password Password item.",
		Fields: []categoryFieldSpec{
			{Attr: "password", FieldID: "password", Title: "password", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true},
		},
		Websites: true,
	},
	{
		TypeName:    "item_software_license",
		Category:    onepassword.ItemCategorySoftwareLicense,
		Description: "Manages a 1Password SoftwareLicense item.",
		DefaultSections: []itemSectionSpec{
			{ID: "customer", Title: "Customer"},
			{ID: "order", Title: "Order"},
			{ID: "publisher", Title: "Publisher"},
		},
		Fields: []categoryFieldSpec{
			{Attr: "version", FieldID: "product_version", Title: "version", Type: onepassword.ItemFieldTypeText},
			{Attr: "license_key", FieldID: "reg_code", Title: "license key", Type: onepassword.ItemFieldTypeText},
			{Attr: "licensed_to", FieldID: "reg_name", Title: "licensed to", Type: onepassword.ItemFieldTypeText, SectionID: "customer"},
			{Attr: "registered_email", FieldID: "reg_email", Title: "registered email", Type: onepassword.ItemFieldTypeEmail, SectionID: "customer"},
			{Attr: "company", FieldID: "company", Title: "company", Type: onepassword.ItemFieldTypeText, SectionID: "customer"},
			{Attr: "download_page", FieldID: "download_link", Title: "download page", Type: onepassword.ItemFieldTypeURL, SectionID: "publisher"},
			{Attr: "publisher", FieldID: "publisher_name", Title: "publisher", Type: onepassword.ItemFieldTypeText, SectionID: "publisher"},
			{Attr: "publisher_website", FieldID: "publisher_website", Title: "website", Type: onepassword.ItemFieldTypeURL, SectionID: "publisher"},
			{Attr: "retail_price", FieldID: "retail_price", Title: "retail price", Type: onepassword.ItemFieldTypeText, SectionID: "publisher"},
			{Attr: "support_email", FieldID: "support_email", Title: "support email", Type: onepassword.ItemFieldTypeEmail, SectionID: "publisher"},
			{Attr: "purchase_date", FieldID: "order_date", Title: "purchase date", Type: onepassword.ItemFieldTypeDate, SectionID: "order"},
			{Attr: "order_number", FieldID: "order_number", Title: "order number", Type: onepassword.ItemFieldTypeText, SectionID: "order"},
			{Attr: "order_total", FieldID: "order_total", Title: "order total", Type: onepassword.ItemFieldTypeText, SectionID: "order"},
		},
	},
	{
		TypeName:    "item_bank_account",
		Category:    onepassword.ItemCategoryBankAccount,
		Description: "Manages a 1Password BankAccount item.",
		DefaultSections: []itemSectionSpec{
			{ID: "branchInfo", Title: "Branch Information"},
		},
		Fields: []categoryFieldSpec{
			{Attr: "bank_name", FieldID: "bankName", Title: "bank name", Type: onepassword.ItemFieldTypeText},
			{Attr: "account_owner", FieldID: "owner", Title: "name on account", Type: onepassword.ItemFieldTypeText},
			{Attr: "account_type", FieldID: "accountType", Title: "type", Type: onepassword.ItemFieldTypeMenu},
			{Attr: "routing_number", FieldID: "routingNo", Title: "routing number", Type: onepassword.ItemFieldTypeText},
			{Attr: "account_number", FieldID: "accountNo", Title: "account number", Type: onepassword.ItemFieldTypeText},
			{Attr: "swift", FieldID: "swift", Title: "SWIFT", Type: onepassword.ItemFieldTypeText},
			{Attr: "iban", FieldID: "iban", Title: "IBAN", Type: onepassword.ItemFieldTypeText},
			{Attr: "pin", FieldID: "telephonePin", Title: "PIN", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true},
			{Attr: "branch_phone", FieldID: "branchPhone", Title: "phone", Type: onepassword.ItemFieldTypePhone, SectionID: "branchInfo"},
			{Attr: "branch_address", FieldID: "branchAddress", Title: "address", Type: onepassword.ItemFieldTypeText, SectionID: "branchInfo"},
		},
	},
	{
		TypeName:    "item_database",
		Category:    onepassword.ItemCategoryDatabase,
		Description: "Manages a 1Password Database item.",
		Fields: []categoryFieldSpec{
			{Attr: "type", FieldID: "database_type", Title: "type", Type: onepassword.ItemFieldTypeMenu},
			{Attr: "hostname", FieldID: "hostname", Title: "server", Type: onepassword.ItemFieldTypeText},
			{Attr: "port", FieldID: "port", Title: "port", Type: onepassword.ItemFieldTypeText},
			{Attr: "database", FieldID: "database", Title: "database", Type: onepassword.ItemFieldTypeText},
			{Attr: "username", FieldID: "username", Title: "username", Type: onepassword.ItemFieldTypeText},
			{Attr: "password", FieldID: "password", Title: "password", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true},
			{Attr: "sid", FieldID: "sid", Title: "SID", Type: onepassword.ItemFieldTypeText},
			{Attr: "alias", FieldID: "alias", Title: "alias", Type: onepassword.ItemFieldTypeText},
			{Attr: "options", FieldID: "options", Title: "connection options", Type: onepassword.ItemFieldTypeText},
		},
	},
	{
		TypeName:    "item_driver_license",
		Category:    onepassword.ItemCategoryDriverLicense,
		Description: "Manages a 1Password DriverLicense item.",
		Fields: []categoryFieldSpec{
			{Attr: "full_name", FieldID: "fullname", Title: "full name", Type: onepassword.ItemFieldTypeText},
			{Attr: "address", FieldID: "address", Title: "address", Type: onepassword.ItemFieldTypeText},
			{Attr: "birth_date", FieldID: "birthdate", Title: "date of birth", Type: onepassword.ItemFieldTypeDate},
			{Attr: "gender", FieldID: "gender", Title: "gender", Type: onepassword.ItemFieldTypeText},
			{Attr: "height", FieldID: "height", Title: "height", Type: onepassword.ItemFieldTypeText},
			{Attr: "number", FieldID: "number", Title: "number", Type: onepassword.ItemFieldTypeText},
			{Attr: "license_class", FieldID: "class", Title: "license class", Type: onepassword.ItemFieldTypeText},
			{Attr: "conditions", FieldID: "conditions", Title: "conditions / restrictions", Type: onepassword.ItemFieldTypeText},
			{Attr: "state", FieldID: "state", Title: "state", Type: onepassword.ItemFieldTypeText},
			{Attr: "country", FieldID: "country", Title: "country", Type: onepassword.ItemFieldTypeText},
			{Attr: "expiry_date", FieldID: "expiry_date", Title: "expiry date", Type: onepassword.ItemFieldTypeMonthYear},
		},
	},
	{
		TypeName:    "item_outdoor_license",
		Category:    onepassword.ItemCategoryOutdoorLicense,
		Description: "Manages a 1Password OutdoorLicense item.",
		Fields: []categoryFieldSpec{
			{Attr: "full_name", FieldID: "name", Title: "full name", Type: onepassword.ItemFieldTypeText},
			{Attr: "valid_from", FieldID: "valid_from", Title: "valid from", Type: onepassword.ItemFieldTypeDate},
			{Attr: "expires", FieldID: "expires", Title: "expires", Type: onepassword.ItemFieldTypeDate},
			{Attr: "approved_wildlife", FieldID: "game", Title: "approved wildlife", Type: onepassword.ItemFieldTypeText},
			{Attr: "maximum_quota", FieldID: "quota", Title: "maximum quota", Type: onepassword.ItemFieldTypeText},
			{Attr: "state", FieldID: "state", Title: "state", Type: onepassword.ItemFieldTypeText},
			{Attr: "country", FieldID: "country", Title: "country", Type: onepassword.ItemFieldTypeText},
		},
	},
	{
		TypeName:    "item_membership",
		Category:    onepassword.ItemCategoryMembership,
		Description: "Manages a 1Password Membership item.",
		Fields: []categoryFieldSpec{
			{Attr: "group", FieldID: "org_name", Title: "group", Type: onepassword.ItemFieldTypeText},
			{Attr: "website", FieldID: "website", Title: "website", Type: onepassword.ItemFieldTypeURL},
			{Attr: "phone", FieldID: "phone", Title: "telephone", Type: onepassword.ItemFieldTypePhone},
			{Attr: "member_name", FieldID: "member_name", Title: "member name", Type: onepassword.ItemFieldTypeText},
			{Attr: "member_since", FieldID: "member_since", Title: "member since", Type: onepassword.ItemFieldTypeMonthYear},
			{Attr: "expiry_date", FieldID: "expiry_date", Title: "expiry date", Type: onepassword.ItemFieldTypeMonthYear},
			{Attr: "member_id", FieldID: "membership_no", Title: "member ID", Type: onepassword.ItemFieldTypeText},
			{Attr: "pin", FieldID: "pin", Title: "PIN", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true},
		},
	},
	{
		TypeName:    "item_passport",
		Category:    onepassword.ItemCategoryPassport,
		Description: "Manages a 1Password Passport item.",
		Fields: []categoryFieldSpec{
			{Attr: "type", FieldID: "type", Title: "type", Type: onepassword.ItemFieldTypeText},
			{Attr: "issuing_country", FieldID: "issuing_country", Title: "issuing country", Type: onepassword.ItemFieldTypeText},
			{Attr: "number", FieldID: "number", Title: "number", Type: onepassword.ItemFieldTypeText},
			{Attr: "full_name", FieldID: "fullname", Title: "full name", Type: onepassword.ItemFieldTypeText},
			{Attr: "gender", FieldID: "gender", Title: "gender", Type: onepassword.ItemFieldTypeText},
			{Attr: "nationality", FieldID: "nationality", Title: "nationality", Type: onepassword.ItemFieldTypeText},
			{Attr: "issuing_authority", FieldID: "issuing_authority", Title: "issuing authority", Type: onepassword.ItemFieldTypeText},
			{Attr: "birth_date", FieldID: "birthdate", Title: "date of birth", Type: onepassword.ItemFieldTypeDate},
			{Attr: "place_of_birth", FieldID: "birthplace", Title: "place of birth", Type: onepassword.ItemFieldTypeText},
			{Attr: "issue_date", FieldID: "issue_date", Title: "issued on", Type: onepassword.ItemFieldTypeDate},
			{Attr: "expiry_date", FieldID: "expiry_date", Title: "expiry date", Type: onepassword.ItemFieldTypeDate},
		},
	},
	{
		TypeName:    "item_rewards",
		Category:    onepassword.ItemCategoryRewards,
		Description: "Manages a 1Password Rewards item.",
		DefaultSections: []itemSectionSpec{
			{ID: "extra", Title: "More Information"},
		},
		Fields: []categoryFieldSpec{
			{Attr: "company_name", FieldID: "company_name", Title: "company name", Type: onepassword.ItemFieldTypeText},
			{Attr: "member_name", FieldID: "member_name", Title: "member name", Type: onepassword.ItemFieldTypeText},
			{Attr: "member_id", FieldID: "membership_no", Title: "member ID", Type: onepassword.ItemFieldTypeText},
			{Attr: "pin", FieldID: "pin", Title: "PIN", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true},
			{Attr: "additional_member_id", FieldID: "additional_no", Title: "member ID (additional)", Type: onepassword.ItemFieldTypeText, SectionID: "extra"},
			{Attr: "member_since", FieldID: "member_since", Title: "member since", Type: onepassword.ItemFieldTypeMonthYear, SectionID: "extra"},
			{Attr: "customer_service_phone", FieldID: "customer_service_phone", Title: "customer service phone", Type: onepassword.ItemFieldTypePhone, SectionID: "extra"},
			{Attr: "reservations_phone", FieldID: "reservations_phone", Title: "phone for reservations", Type: onepassword.ItemFieldTypePhone, SectionID: "extra"},
			{Attr: "website", FieldID: "website", Title: "website", Type: onepassword.ItemFieldTypeURL, SectionID: "extra"},
		},
	},
	{
		TypeName:    "item_social_security_number",
		Category:    onepassword.ItemCategorySocialSecurityNumber,
		Description: "Manages a 1Password SocialSecurityNumber item.",
		Fields: []categoryFieldSpec{
			{Attr: "name", FieldID: "name", Title: "name", Type: onepassword.ItemFieldTypeText},
			{Attr: "number", FieldID: "number", Title: "number", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true},
		},
	},
	{
		TypeName:    "item_router",
		Category:    onepassword.ItemCategoryRouter,
		Description: "Manages a 1Password Router item.",
		Fields: []categoryFieldSpec{
			{Attr: "base_station_name", FieldID: "name", Title: "base station name", Type: onepassword.ItemFieldTypeText},
			{Attr: "base_station_password", FieldID: "password", Title: "base station password", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true},
			{Attr: "server", FieldID: "server", Title: "server / IP address", Type: onepassword.ItemFieldTypeText},
			{Attr: "airport_id", FieldID: "airport_id", Title: "AirPort ID", Type: onepassword.ItemFieldTypeText},
			{Attr: "network_name", FieldID: "network_name", Title: "network name", Type: onepassword.ItemFieldTypeText},
			{Attr: "wireless_security", FieldID: "wireless_security", Title: "wireless security", Type: onepassword.ItemFieldTypeMenu},
			{Attr: "wireless_password", FieldID: "wireless_password", Title: "wireless network password", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true},
			{Attr: "disk_password", FieldID: "disk_password", Title: "attached storage password", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true},
		},
	},
	{
		TypeName:    "item_server",
		Category:    onepassword.ItemCategoryServer,
		Description: "Manages a 1Password Server item.",
		DefaultSections: []itemSectionSpec{
			{ID: "admin_console", Title: "Admin Console"},
			{ID: "hosting_provider_details", Title: "Hosting Provider"},
		},
		Fields: []categoryFieldSpec{
			{Attr: "url", FieldID: "url", Title: "URL", Type: onepassword.ItemFieldTypeText},
			{Attr: "username", FieldID: "username", Title: "username", Type: onepassword.ItemFieldTypeText},
			{Attr: "password", FieldID: "password", Title: "password", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true},
			{Attr: "admin_console_url", FieldID: "admin_console_url", Title: "admin console URL", Type: onepassword.ItemFieldTypeText, SectionID: "admin_console"},
			{Attr: "admin_console_username", FieldID: "admin_console_username", Title: "admin console username", Type: onepassword.ItemFieldTypeText, SectionID: "admin_console"},
			{Attr: "admin_console_password", FieldID: "admin_console_password", Title: "console password", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true, SectionID: "admin_console"},
			{Attr: "hosting_provider_name", FieldID: "name", Title: "name", Type: onepassword.ItemFieldTypeText, SectionID: "hosting_provider_details"},
			{Attr: "hosting_provider_website", FieldID: "website", Title: "website", Type: onepassword.ItemFieldTypeText, SectionID: "hosting_provider_details"},
			{Attr: "support_url", FieldID: "support_contact_url", Title: "support URL", Type: onepassword.ItemFieldTypeText, SectionID: "hosting_provider_details"},
			{Attr: "support_phone", FieldID: "support_contact_phone", Title: "support phone", Type: onepassword.ItemFieldTypeText, SectionID: "hosting_provider_details"},
		},
	},
	{
		TypeName:    "item_email",
		Category:    onepassword.ItemCategoryEmail,
		Description: "Manages a 1Password Email item.",
		// The template's section IDs contain spaces and capitals ("Contact
		// Information", "SMTP"); they are copied verbatim.
		DefaultSections: []itemSectionSpec{
			{ID: "Contact Information", Title: "Contact Information"},
			{ID: "SMTP", Title: "SMTP"},
		},
		Fields: []categoryFieldSpec{
			{Attr: "type", FieldID: "pop_type", Title: "type", Type: onepassword.ItemFieldTypeMenu},
			{Attr: "username", FieldID: "pop_username", Title: "username", Type: onepassword.ItemFieldTypeText},
			{Attr: "server", FieldID: "pop_server", Title: "server", Type: onepassword.ItemFieldTypeText},
			{Attr: "port", FieldID: "pop_port", Title: "port number", Type: onepassword.ItemFieldTypeText},
			{Attr: "password", FieldID: "pop_password", Title: "password", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true},
			{Attr: "security", FieldID: "pop_security", Title: "security", Type: onepassword.ItemFieldTypeMenu},
			{Attr: "auth_method", FieldID: "pop_authentication", Title: "auth method", Type: onepassword.ItemFieldTypeMenu},
			{Attr: "smtp_server", FieldID: "smtp_server", Title: "SMTP server", Type: onepassword.ItemFieldTypeText, SectionID: "SMTP"},
			{Attr: "smtp_port", FieldID: "smtp_port", Title: "port number", Type: onepassword.ItemFieldTypeText, SectionID: "SMTP"},
			{Attr: "smtp_username", FieldID: "smtp_username", Title: "username", Type: onepassword.ItemFieldTypeText, SectionID: "SMTP"},
			{Attr: "smtp_password", FieldID: "smtp_password", Title: "password", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true, SectionID: "SMTP"},
			{Attr: "smtp_security", FieldID: "smtp_security", Title: "security", Type: onepassword.ItemFieldTypeMenu, SectionID: "SMTP"},
			{Attr: "smtp_auth_method", FieldID: "smtp_authentication", Title: "auth method", Type: onepassword.ItemFieldTypeMenu, SectionID: "SMTP"},
			// HCL attr is email_provider because Terraform reserves
			// "provider" as a root attribute name; the canonical 1Password
			// field ID stays "provider".
			{Attr: "email_provider", FieldID: "provider", Title: "provider", Type: onepassword.ItemFieldTypeText, SectionID: "Contact Information"},
			{Attr: "provider_website", FieldID: "provider_website", Title: "provider's website", Type: onepassword.ItemFieldTypeText, SectionID: "Contact Information"},
			{Attr: "phone_local", FieldID: "phone_local", Title: "phone (local)", Type: onepassword.ItemFieldTypeText, SectionID: "Contact Information"},
			{Attr: "phone_toll_free", FieldID: "phone_tollfree", Title: "phone (toll free)", Type: onepassword.ItemFieldTypeText, SectionID: "Contact Information"},
		},
	},
	{
		TypeName:    "item_api_credentials",
		Category:    onepassword.ItemCategoryAPICredentials,
		Description: "Manages a 1Password ApiCredentials item.",
		Fields: []categoryFieldSpec{
			{Attr: "username", FieldID: "username", Title: "username", Type: onepassword.ItemFieldTypeText},
			{Attr: "credential", FieldID: "credential", Title: "credential", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true},
			{Attr: "type", FieldID: "type", Title: "type", Type: onepassword.ItemFieldTypeMenu},
			{Attr: "filename", FieldID: "filename", Title: "filename", Type: onepassword.ItemFieldTypeText},
			{Attr: "valid_from", FieldID: "validFrom", Title: "valid from", Type: onepassword.ItemFieldTypeDate},
			{Attr: "expires", FieldID: "expires", Title: "expires", Type: onepassword.ItemFieldTypeDate},
			{Attr: "hostname", FieldID: "hostname", Title: "hostname", Type: onepassword.ItemFieldTypeText},
		},
		Websites: true,
	},
	{
		TypeName:    "item_medical_record",
		Category:    onepassword.ItemCategoryMedicalRecord,
		Description: "Manages a 1Password MedicalRecord item.",
		DefaultSections: []itemSectionSpec{
			{ID: "medication", Title: "medication"},
		},
		Fields: []categoryFieldSpec{
			{Attr: "date", FieldID: "date", Title: "date", Type: onepassword.ItemFieldTypeDate},
			{Attr: "location", FieldID: "location", Title: "location", Type: onepassword.ItemFieldTypeText},
			{Attr: "healthcare_professional", FieldID: "healthcareprofessional", Title: "healthcare professional", Type: onepassword.ItemFieldTypeText},
			{Attr: "patient", FieldID: "patient", Title: "patient", Type: onepassword.ItemFieldTypeText},
			{Attr: "reason", FieldID: "reason", Title: "reason for visit", Type: onepassword.ItemFieldTypeText},
			{Attr: "medication", FieldID: "medication", Title: "medication", Type: onepassword.ItemFieldTypeText, SectionID: "medication"},
			{Attr: "dosage", FieldID: "dosage", Title: "dosage", Type: onepassword.ItemFieldTypeText, SectionID: "medication"},
			{Attr: "medication_notes", FieldID: "notes", Title: "medication notes", Type: onepassword.ItemFieldTypeText, SectionID: "medication"},
		},
	},
	{
		TypeName:    "item_ssh_key",
		Category:    onepassword.ItemCategorySSHKey,
		Description: "Manages a 1Password SshKey item.",
		Fields: []categoryFieldSpec{
			{Attr: "private_key", FieldID: "private_key", Title: "private key", Type: onepassword.ItemFieldTypeSSHKey, Sensitive: true},
		},
		SSHDetails: []sshDetailSpec{
			{Attr: "public_key", FieldID: "private_key", DetailKey: "public_key"},
			{Attr: "fingerprint", FieldID: "private_key", DetailKey: "fingerprint"},
			{Attr: "key_type", FieldID: "private_key", DetailKey: "key_type"},
		},
		GenerateSSHKey: true,
	},
	{
		TypeName:    "item_crypto_wallet",
		Category:    onepassword.ItemCategoryCryptoWallet,
		Description: "Manages a 1Password CryptoWallet item.",
		DefaultSections: []itemSectionSpec{
			{ID: "wallet", Title: "Wallet"},
		},
		Fields: []categoryFieldSpec{
			{Attr: "recovery_phrase", FieldID: "recoveryPhrase", Title: "recovery phrase", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true},
			{Attr: "password", FieldID: "password", Title: "password", Type: onepassword.ItemFieldTypeConcealed, Sensitive: true},
			{Attr: "wallet_address", FieldID: "walletAddress", Title: "wallet address", Type: onepassword.ItemFieldTypeText, SectionID: "wallet"},
		},
	},
}

// fieldDescription renders the schema description of a typed field.
func fieldDescription(f categoryFieldSpec) string {
	if f.Address {
		return fmt.Sprintf("%s (address details).", f.Title)
	}
	switch f.Type {
	case onepassword.ItemFieldTypeDate:
		return fmt.Sprintf("%s. Must be `YYYY-MM-DD`.", f.Title)
	case onepassword.ItemFieldTypeMonthYear:
		return fmt.Sprintf("%s. Must be `MM/YYYY`.", f.Title)
	case onepassword.ItemFieldTypeSSHKey:
		return fmt.Sprintf("%s. Must be a decrypted PEM private key.", f.Title)
	}
	return fmt.Sprintf("%s.", f.Title)
}
