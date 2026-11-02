package badge

// DotClass returns the classes for a small status dot of variant v, using the
// same color tokens as the badge of that variant. VariantOutline is a hollow
// dot; the zero value is the primary dot. timeline and nodepill draw their
// dots with it, so a dot of one variant looks the same everywhere.
func DotClass(v Variant) string {
	switch v {
	case VariantDestructive:
		return "bg-destructive"
	case VariantSuccess:
		return "bg-success"
	case VariantWarning:
		return "bg-warning"
	case VariantInfo:
		return "bg-info"
	case VariantSecondary:
		return "bg-muted-foreground"
	case VariantOutline:
		return "border-2 border-border bg-background"
	default:
		return "bg-primary"
	}
}
