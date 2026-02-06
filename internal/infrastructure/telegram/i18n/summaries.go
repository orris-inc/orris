package i18n

// Summary messages (daily/weekly) are built inline in the use cases
// (senddailysummary.go and sendweeklysummary.go) because they depend
// on application-layer DTO types and are tightly coupled to data
// gathering logic. Both use cases accept i18n.Lang for per-binding
// language-aware message generation.
