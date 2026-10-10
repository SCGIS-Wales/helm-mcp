package helmengine

// InstallOptionsForUpgrade maps upgrade options onto the install the Helm
// CLI performs for `helm upgrade --install` when the release does not exist
// (or its last revision was uninstalled, in which case replace is set). The
// field list follows the CLI's own copy in pkg/cmd/upgrade.go.
//
// The upgrade action ignores its Install field: the fallback lives in the
// CLI, so each engine has to perform it.
func InstallOptionsForUpgrade(u *UpgradeOptions, replace bool) *InstallOptions {
	return &InstallOptions{
		ChartSourceOptions:       u.ChartSourceOptions,
		ValueSetOptions:          u.ValueSetOptions,
		PostRenderOptions:        u.PostRenderOptions,
		ReleaseName:              u.ReleaseName,
		Chart:                    u.Chart,
		Version:                  u.Version,
		Values:                   u.Values,
		ValuesFiles:              u.ValuesFiles,
		CreateNamespace:          u.CreateNamespace,
		Wait:                     u.Wait,
		WaitForJobs:              u.WaitForJobs,
		Timeout:                  u.Timeout,
		DryRun:                   u.DryRun,
		Description:              u.Description,
		DisableHooks:             u.DisableHooks,
		Replace:                  replace,
		SkipCRDs:                 u.SkipCRDs,
		DependencyUpdate:         u.DependencyUpdate,
		Labels:                   u.Labels,
		RollbackOnFailure:        u.RollbackOnFailure,
		Devel:                    u.Devel,
		SubNotes:                 u.SubNotes,
		HideNotes:                u.HideNotes,
		SkipSchemaValidation:     u.SkipSchemaValidation,
		DisableOpenAPIValidation: u.DisableOpenAPIValidation,
		EnableDNS:                u.EnableDNS,
		ForceReplace:             u.Force,
		ServerSideApply:          u.ServerSideApply,
		TakeOwnership:            u.TakeOwnership,
		HideSecret:               u.HideSecret,
		ForceConflicts:           u.ForceConflicts,
		WaitStrategy:             u.WaitStrategy,
	}
}
