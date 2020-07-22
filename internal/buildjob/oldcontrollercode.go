package buildjob

// NOTE: taken from original Reconcile func
//spec := build.Spec
//
//// ignore resources that have been processed on start
//if build.Status.State != "" {
//	log.Info("Skipping resource", "state", build.Status.State)
//	return result, nil
//}
//
////log = log.WithValues("annotations", build.Annotations)
//
//// mark resource status and update before launching build
//build.Status.SetState(forgev1alpha1.Building)
//build.Status.BuildStartedAt = &metav1.Time{Time: time.Now()}
//
//if err := r.updateResourceStatus(ctx, log, &build); err != nil {
//	return result, err
//}
//
//// process registry configs
//registries, err := r.buildRegistryConfig(ctx, spec.Registries)
//if err != nil {
//	log.Error(err, "Registry config processing failed")
//
//	build.Status.State = forgev1alpha1.Failed
//	build.Status.ErrorMessage = err.Error()
//
//	if iErr := r.updateResourceStatus(ctx, log, &build); iErr != nil {
//		return result, iErr
//	}
//	return result, nil
//}
//
//// construct build directives
//opts := &config.BuildOptions{
//	Registries:     registries,
//	PushRegistries: spec.PushRegistries,
//	ImageName:      spec.ImageName,
//	ImageSizeLimit: spec.ImageSizeLimit,
//	ContextURL:     spec.Context,
//	NoCache:        spec.NoCache,
//	Labels:         spec.Labels,
//	BuildArgs:      spec.BuildArgs,
//	CpuQuota:       spec.CpuQuota,
//	Memory:         spec.Memory,
//	PluginData:     spec.PluginData,
//	Timeout:        time.Duration(build.Spec.TimeoutSeconds) * time.Second,
//}
//
//log.Info(fmt.Sprintf("Building image %s and pushing to: %s", spec.ImageName, strings.Join(spec.PushRegistries, ", ")))
//log.Info(strings.Repeat("=", 70))
//
//// dispatch build operation
//r.Builder.SetLogger(log)
//imageURLs, err := r.Builder.BuildAndPush(ctx, opts)
//
//log.Info(strings.Repeat("=", 70))
//
//if err != nil {
//	log.V(1).Info("received error during build and push", "error", err)
//
//	if !logUnknownInstructionError(log, err) {
//		log.Info(fmt.Sprintf("Error during image build and push: %s", err.Error()))
//	}
//
//	build.Status.SetState(forgev1alpha1.Failed)
//	build.Status.ErrorMessage = err.Error()
//
//	if uErr := r.updateResourceStatus(ctx, log, &build); uErr != nil {
//		err = errors.Wrapf(uErr, "status update failed: triggered by %v", err)
//		return result, err
//	}
//
//	return result, nil
//}
//
//// mark resource status to indicate build was successful
//build.Status.ImageURLs = imageURLs
//build.Status.SetState(forgev1alpha1.Completed)
//build.Status.BuildCompletedAt = &metav1.Time{Time: time.Now()}
//
//if err := r.updateResourceStatus(ctx, log, &build); err != nil {
//	return result, err
//}
//
//// reconcile result will ensure this event is not enqueued again
//return result, nil

///*
//	OLDER CODE BELOW
//*/
//
//// TODO: move the following 2 funcs off of the Reconciler and give them their own k8s client
//
//func (r *ContainerImageBuildReconciler) buildRegistryConfig(ctx context.Context, apiRegs []forgev1alpha1.Registry) ([]config.Registry, error) {
//	var configs []config.Registry
//	for _, apiReg := range apiRegs {
//		regConf := config.Registry{
//			Host:   apiReg.Server,
//			NonSSL: apiReg.NonSSL,
//		}
//
//		// NOTE: move BasicAuth validation into an admission webhook at a later time
//		if err := apiReg.BasicAuth.Validate(); err != nil {
//			return nil, errors.Wrap(err, "basic auth validation failed")
//		}
//
//		switch {
//		case apiReg.BasicAuth.IsInline():
//			regConf.Username = apiReg.BasicAuth.Username
//			regConf.Password = apiReg.BasicAuth.Password
//		case apiReg.BasicAuth.IsSecret():
//			var err error
//			regConf.Username, regConf.Password, err = r.getDockerAuthFromSecret(ctx, apiReg.Server, apiReg.BasicAuth.SecretName, apiReg.BasicAuth.SecretNamespace)
//			if err != nil {
//				return nil, err
//			}
//		}
//
//		configs = append(configs, regConf)
//	}
//
//	return configs, nil
//}
//
//func (r *ContainerImageBuildReconciler) getDockerAuthFromSecret(ctx context.Context, host, name, namespace string) (string, string, error) {
//	var secret corev1.Secret
//	if err := r.Client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &secret); err != nil {
//		return "", "", errors.Wrap(err, "cannot find registry auth secret")
//	}
//
//	if secret.Type != corev1.SecretTypeDockerConfigJson {
//		return "", "", fmt.Errorf("registry auth secret must be %v, not %v", corev1.SecretTypeDockerConfigJson, secret.Type)
//	}
//
//	input := secret.Data[corev1.DockerConfigJsonKey]
//	var output credentials.DockerConfigJSON
//	if err := json.Unmarshal(input, &output); err != nil {
//		return "", "", errors.Wrap(err, "cannot parse docker config in registry secret")
//	}
//
//	auth, ok := output.Auths[host]
//	if !ok {
//		var urls []string
//		for k, _ := range output.Auths {
//			urls = append(urls, k)
//		}
//		return "", "", fmt.Errorf("registry server %q is not in registry secret %q: server list %v", host, name, urls)
//	}
//
//	return auth.Username, auth.Password, nil
//}
//
//func (r *ContainerImageBuildReconciler) updateResourceStatus(ctx context.Context, log logr.Logger, build *forgev1alpha1.ContainerImageBuild) error {
//	if err := r.Status().Update(ctx, build); err != nil {
//		log.Error(err, "Unable to update status")
//
//		msg := fmt.Sprintf("Forge was unable to update this resource status: %v", err)
//		r.Recorder.Event(build, corev1.EventTypeWarning, "UpdateFailed", msg)
//
//		return err
//	}
//
//	if r.Producer != nil {
//		update := &StatusUpdate{
//			Name:          build.Name,
//			Annotations:   build.Annotations,
//			ObjectLink:    strings.TrimSuffix(build.GetSelfLink(), "/status"),
//			PreviousState: string(build.Status.PreviousState),
//			CurrentState:  string(build.Status.State),
//			ImageURLs:     build.Status.ImageURLs,
//			ErrorMessage:  build.Status.ErrorMessage,
//		}
//		if err := r.Producer.Publish(update); err != nil {
//			log.Error(err, "Unable to publish message")
//		}
//	}
//	return nil
//}
//
//// This logs the underlying error from a build when the display channels inside builder.embedded have not yet been initialized.
//func logUnknownInstructionError(log logr.Logger, err error) bool {
//	if unwrappedError := errors.Unwrap(err); unwrappedError != nil {
//		err = unwrappedError
//	}
//	cause := errors.Cause(err)
//	if instructions.IsUnknownInstruction(cause) {
//		log.Error(err, cause.Error())
//		return true
//	}
//
//	return false
//}
