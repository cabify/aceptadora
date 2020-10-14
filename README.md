# aceptadora

Aceptadora provides the boilerplate to orchestrate the containers around an acceptance test.

Aceptadora is a replacement for `docker-compose` in acceptance tests, and it also allows running and debugging tests from your IDE.

The acceptance tests of this package are an example of usage for this package.

# Motivations

We've been using `docker-compose` for acceptance tests for a long time, and while this approach did have issues[^1] we lived with it until the introduction of gRPC. 
Once we started playing with gRPC we found an issue: the gRPC golang client tries to connect to the service when the service is started, and if it fails, it will keep returning that error for a while. 
Since we started our test subjects before the acceptance-tester in the former approach, our test subjects failed to call the mocked gRPC servers on the acceptance-tester.

So, we created a library that restarted our test subject interacting with `docker` from the acceptance test itself.

Then, we found the need to restart multiple test subjects plus the complexity of having to handle two kinds of configurations (`docker-compose` for dependencies and `aceptadora` for test subjects) so we decided to extend the functionality of `aceptadora` to completely replace the `docker-compose` and allow managing the lifecycle of all dependencies from the test.

# Decisions

Everything in aceptadora accepts `t *testing.T` and everything does `require.NoError(t, err)` because in tests nobody's going to handle the errors anyway, so we apply a fail-fast strategy, removing the retured errors and keeping the API clean for clearer acceptance tests.

# Running

In order to handle multiple environments there are some stages in config loading. Notice that all the configs loaded expand the env vars set by `${VAR}` to their values from what's already loaded.

First, we load some very basic env-dependant config, deciding on env vars to load a local or gitlab config (we're using `darwin` instead of `local` since someone may want to init a `linux` config too). 
This configuration mostly provides details about networking setup:
- Where can acceptance-tester reach the services? On Mac this would be `localhost` but on Gitlab it's `docker`
- Where can services reach the acceptance-tester? On Mac it's `docker.for.mac.localhost`, on Linux it would be `localhost` and on Gitlab it's the IP address of the docker running the test itself. 
  In order to evaluate the IP address of the test runner on Gitlab we perform some previous checks in `scripts/acceptance.sh`.
  
One may wonder: why don't we just decide all of that in `scripts/acceptance.sh`? 
The answer is that deciding that from the test itself allows us running the tests from any IDE as a normal test, instead of having a proxy script. 
Of course loading some env vars would work, but since `aceptadora` can do this for you, why should we care?

Then we load more env configs for the test itself, usually `acceptance.env`, which tells the acceptance test where the `aceptadora.yml` file is located, and how images from different docker registries are pulled.
Notice that `acceptance.env` can be specific to each suite you may have if their paths are different. 
You can load as many common configs as you want, loading an `acceptance.env` and then `../config/acceptance.env` for example.

Here `aceptadora.New()` would load the `aceptadora.yml` file from the provided dir and filename, and it will also set `${YAMLDIR}` variable for the yamls itself to be able to reference its configs and binds from where the yaml is located instead of from where the test is located: this allows us having multiple test suites in different folders using the same `aceptadora.yml` file.

Finally, we run services by just running `aceptadora.Run(ctx, "svc-name-in-the-yaml")`.

Aceptadora will also take care of stopping the services, you can call `aceptadora.Stop(ctx, svcName)` to stop one of them, or `StopAll(ctx)` to stop all the (still running) services.

[^1]: With `docker-compose` approach, test subjects and dependencies were not restarted between tests, so we had to have some suites depending on other ones, which made the tests flaky and slow.