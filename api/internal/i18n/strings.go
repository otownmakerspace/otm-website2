package i18n

// Strings is the set of UI strings rendered by the backend's htmx fragments
// and inline error messages. One instance per language; selected by For(lang).
//
// Keep keys grouped by surface (membership, profile, password, billing) so
// translators can see context. Adding a new field requires:
//   1. Add the field to this struct.
//   2. Add the value to both en and da below.
//   3. Reference it in the relevant template as {{ .S.<Field> }}.
type Strings struct {
	// ── Hero (hero.html) ──
	HeroGreetingNamed     string // "Hi %s" — printf-style with first name
	HeroGreetingAnonymous string // "Welcome back" — fallback when name is empty
	HeroRenewsOn          string // "Membership renews %s" — printf with formatted date
	HeroEnding            string // "Membership ends %s"
	HeroInactive          string // "No active membership"
	HeroOpenHouse         string // "See you Thursday %s at 17:00." — printf

	// ── Account menu (dashboard layout) ──
	AccountMenu string // button label "Account"
	BackToSite  string // dropdown item
	SignOut     string // dropdown item

	// ── Membership card (subscriptions.html) ──
	Membership          string
	StatusActive        string
	Plan                string
	NextRenewal         string
	MemberSince         string
	StartedOn           string
	OpenHouseHintTpl    string // printf template "See you at the next Open House — %s, 17:00."
	CancelMembership    string
	CancelEndsOn        string // "Your membership is set to end on {date}. You'll keep access until then."
	NoActiveMembership  string
	NoActiveBody        string
	BecomeMemberAgain   string
	MembershipCancelled string

	// ── Cancel modal (cancel-modal.html) ──
	CancelModalTitle       string
	CancelModalKeepAccess  string // printf "You'll keep access until %s."
	CancelModalBelongings  string
	CancelModalKeep        string // "Keep membership"
	CancelModalConfirm     string // "End membership"
	CancelModalConfirmDated string // printf "End on %s"

	// ── Settings group label (dashboard layout) ──
	SettingsSection string

	// ── Profile card (profile.html) ──
	Profile          string
	Saved            string
	FullName         string
	Email            string
	EmailReadOnly    string
	Phone            string
	Save             string
	Cancel           string
	EditProfile      string
	ErrNameEmpty     string
	ErrTooLong       string
	ErrSaveFailed    string

	// ── Password card (password.html) ──
	ChangePassword     string
	Updated            string
	CurrentPassword    string
	NewPassword        string
	ConfirmNewPassword string
	PasswordHint       string // "8-31 characters."
	UpdatePassword     string
	ChangeAgain        string
	PasswordUpdatedMsg string
	ErrAllRequired     string
	ErrPwMismatch      string
	ErrPwLength        string
	ErrPwSame          string
	ErrCurrentWrong    string
	ErrPwSaveFailed    string
	ErrVerifyFailed    string

	// ── Billing card (invoices.html) ──
	Billing           string
	ManagePayment     string
	NoInvoicesYet     string
	InvoiceStatusPaid string
	InvoiceStatusOpen string
	InvoicePDF        string
	InvoiceView       string
	InvoiceTitleMonth string // printf template, e.g. "%s membership" → "May 2026 membership"

	// ── Price selector (prices.html, fragment in /checkout/) ──
	SelectMembership  string
	NoPricesAvailable string
}

var en = Strings{
	HeroGreetingNamed:     "Hi %s",
	HeroGreetingAnonymous: "Welcome back",
	HeroRenewsOn:          "Membership renews %s",
	HeroEnding:            "Membership ends %s",
	HeroInactive:          "No active membership",
	HeroOpenHouse:         "Next Open House: %s at 17:00.",

	AccountMenu: "Account",
	BackToSite:  "Back to site",
	SignOut:     "Sign out",

	Membership:          "Membership",
	StatusActive:        "Active",
	Plan:                "Plan",
	NextRenewal:         "Next renewal",
	MemberSince:         "Member since",
	StartedOn:           "Member since",
	OpenHouseHintTpl:    "See you at the next Open House — %s, 17:00",
	CancelMembership:    "Cancel membership",
	CancelEndsOn:        "Your membership is set to end on %s. You'll keep access until then.",
	NoActiveMembership:  "No active membership",
	NoActiveBody:        "You don't have an active membership right now.",
	BecomeMemberAgain:   "Become a member again",
	MembershipCancelled: "Membership cancelled.",

	CancelModalTitle:        "Cancel your membership?",
	CancelModalKeepAccess:   "You'll keep access to the makerspace until %s — the end of the period you've already paid for.",
	CancelModalBelongings:   "After that, you'll need to pick up any belongings stored at the makerspace.",
	CancelModalKeep:         "Keep membership",
	CancelModalConfirm:      "End membership",
	CancelModalConfirmDated: "End on %s",

	SettingsSection: "Account settings",

	Profile:        "Profile",
	Saved:          "Saved",
	FullName:       "Full name",
	Email:          "Email",
	EmailReadOnly:  "Contact us if you need to change your email.",
	Phone:          "Phone",
	Save:           "Save",
	Cancel:         "Cancel",
	EditProfile:    "Edit profile",
	ErrNameEmpty:   "Name can't be empty.",
	ErrTooLong:     "Name or phone is too long.",
	ErrSaveFailed:  "Couldn't save. Try again.",

	ChangePassword:     "Change password",
	Updated:            "Updated",
	CurrentPassword:    "Current password",
	NewPassword:        "New password",
	ConfirmNewPassword: "Confirm new password",
	PasswordHint:       "8–31 characters.",
	UpdatePassword:     "Update password",
	ChangeAgain:        "Change again",
	PasswordUpdatedMsg: "Your password has been updated. Use it next time you sign in.",
	ErrAllRequired:     "All fields are required.",
	ErrPwMismatch:      "New passwords don't match.",
	ErrPwLength:        "New password must be %d-%d characters.",
	ErrPwSame:          "New password must be different from the current one.",
	ErrCurrentWrong:    "Current password is incorrect.",
	ErrPwSaveFailed:    "Couldn't save the new password. Try again.",
	ErrVerifyFailed:    "Couldn't verify your account. Try again.",

	Billing:           "Billing",
	ManagePayment:     "Manage payment & invoices →",
	NoInvoicesYet:     "No invoices yet.",
	InvoiceStatusPaid: "paid",
	InvoiceStatusOpen: "open",
	InvoicePDF:        "PDF",
	InvoiceView:       "View",
	InvoiceTitleMonth: "%s membership",

	SelectMembership:  "Choose your membership",
	NoPricesAvailable: "No memberships are available right now. Please try again later.",
}

var da = Strings{
	HeroGreetingNamed:     "Hej %s",
	HeroGreetingAnonymous: "Velkommen tilbage",
	HeroRenewsOn:          "Medlemskab fornys %s",
	HeroEnding:            "Medlemskab udløber %s",
	HeroInactive:          "Intet aktivt medlemskab",
	HeroOpenHouse:         "Næste Open House: %s kl. 17:00.",

	AccountMenu: "Konto",
	BackToSite:  "Tilbage til hjemmesiden",
	SignOut:     "Log ud",

	Membership:          "Medlemskab",
	StatusActive:        "Aktivt",
	Plan:                "Abonnement",
	NextRenewal:         "Næste fornyelse",
	MemberSince:         "Medlem siden",
	StartedOn:           "Medlem siden",
	OpenHouseHintTpl:    "Vi ses til næste Open House — %s kl. 17:00",
	CancelMembership:    "Opsig medlemskab",
	CancelEndsOn:        "Dit medlemskab udløber %s. Du beholder adgang indtil da.",
	NoActiveMembership:  "Intet aktivt medlemskab",
	NoActiveBody:        "Du har ikke et aktivt medlemskab lige nu.",
	BecomeMemberAgain:   "Bliv medlem igen",
	MembershipCancelled: "Medlemskab opsagt.",

	CancelModalTitle:        "Opsig dit medlemskab?",
	CancelModalKeepAccess:   "Du beholder adgang til værkstedet indtil %s — slutningen af den periode du allerede har betalt for.",
	CancelModalBelongings:   "Derefter skal du hente eventuelle ejendele opbevaret på værkstedet.",
	CancelModalKeep:         "Behold medlemskab",
	CancelModalConfirm:      "Opsig medlemskab",
	CancelModalConfirmDated: "Opsig pr. %s",

	SettingsSection: "Kontoindstillinger",

	Profile:        "Profil",
	Saved:          "Gemt",
	FullName:       "Fulde navn",
	Email:          "Email",
	EmailReadOnly:  "Kontakt os hvis du skal ændre din email.",
	Phone:          "Telefon",
	Save:           "Gem",
	Cancel:         "Annuller",
	EditProfile:    "Rediger profil",
	ErrNameEmpty:   "Navn må ikke være tomt.",
	ErrTooLong:     "Navn eller telefon er for lange.",
	ErrSaveFailed:  "Kunne ikke gemme. Prøv igen.",

	ChangePassword:     "Skift adgangskode",
	Updated:            "Opdateret",
	CurrentPassword:    "Nuværende adgangskode",
	NewPassword:        "Ny adgangskode",
	ConfirmNewPassword: "Bekræft ny adgangskode",
	PasswordHint:       "8–31 tegn.",
	UpdatePassword:     "Opdater adgangskode",
	ChangeAgain:        "Skift igen",
	PasswordUpdatedMsg: "Din adgangskode er opdateret. Brug den næste gang du logger ind.",
	ErrAllRequired:     "Alle felter er påkrævede.",
	ErrPwMismatch:      "De nye adgangskoder stemmer ikke overens.",
	ErrPwLength:        "Ny adgangskode skal være %d-%d tegn.",
	ErrPwSame:          "Ny adgangskode skal være forskellig fra den nuværende.",
	ErrCurrentWrong:    "Nuværende adgangskode er forkert.",
	ErrPwSaveFailed:    "Kunne ikke gemme den nye adgangskode. Prøv igen.",
	ErrVerifyFailed:    "Kunne ikke verificere din konto. Prøv igen.",

	Billing:           "Betaling",
	ManagePayment:     "Administrer betaling og fakturaer →",
	NoInvoicesYet:     "Ingen fakturaer endnu.",
	InvoiceStatusPaid: "betalt",
	InvoiceStatusOpen: "åben",
	InvoicePDF:        "PDF",
	InvoiceView:       "Vis",
	InvoiceTitleMonth: "Medlemskab — %s",

	SelectMembership:  "Vælg dit medlemskab",
	NoPricesAvailable: "Der er ingen medlemskaber tilgængelige lige nu. Prøv igen senere.",
}

// For returns the Strings instance for the given language.
// Unknown languages fall back to English.
func For(lang Lang) Strings {
	if lang == DA {
		return da
	}
	return en
}
