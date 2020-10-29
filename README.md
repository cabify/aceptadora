# aceptadora

![lint](https://github.com/cabify/aceptadora/workflows/lint/badge.svg)
![acceptance](https://github.com/cabify/aceptadora/workflows/acceptance/badge.svg)
[![Travis build status](https://travis-ci.com/cabify/aceptadora.svg?branch=master)](https://travis-ci.com/cabify/aceptadora)

Aceptadora provides the boilerplate to orchestrate the containers for an acceptance test.

Aceptadora is a replacement for `docker-compose` in acceptance tests, and it also allows running and debugging tests from your IDE.

# Example

The [acceptance tests](./acceptance/suite/acceptance_suite_test.go) of this package are an example of usage for this package. There are also CI builds running on both [Github Actions](.github/workflows/acceptance.yml) and [Travis CI](./.travis.yml) that serve as an example.

Long story short:
 - Define your service in a YAML file inspired by docker-compose format, with an idiomatic name of [`aceptadora.yml`](./acceptance/aceptadora.yml)
 - Define your environment variables in [some `config.env` files](./acceptance/config.env)
 - Load them in your test:
 ```go
	aceptadora.SetEnv(
		s.T(),
		aceptadora.OneOfEnvConfigs(
			aceptadora.EnvConfigWhenEnvVarPresent("../config/gitlab.env", "GITLAB_CI"),
			aceptadora.EnvConfigAlways("../config/default.env"),
		),
		aceptadora.EnvConfigAlways("acceptance.env"),
	) 
 ```
 - Fill the [`aceptadora.Config`](./aceptadora.go) values, you can use `github.com/colega/envconfig` for that
 ```go
	envconfig.MustProcess("ACCEPTANCE", &s.cfg)
 ``` 
 - Instantiate the _aceptadora_:
 ```go
	aceptadora := aceptadora.New(t, cfg)
 ```
 - Start your service:
 ```go
	aceptadora.Run(ctx, "redis")
 ```
 - Test stuff
 - When you're done, stop it using `aceptadora.StopAll(ctx)` or `aceptadora.Stop(ctx, "redis")`

# Motivations

We've been using `docker-compose` for acceptance tests for a long time, and while this approach did have issues (since test subjects and dependencies were not restarted between tests, we had to have some suites depending on other ones, which made the tests flaky and slow) we lived with it until the introduction of gRPC. 
Once we started playing with gRPC we found an issue: the gRPC golang client tries to connect to the service when the service is started, and if it fails, it will keep returning that error for a while. 
Since we started our test subjects before the acceptance-tester in the former approach, our test subjects failed to call the mocked gRPC servers on the acceptance-tester.

So, we created a library that restarted our test subject interacting with `docker` from the acceptance test itself.

Then, we found the need to restart multiple test subjects plus the complexity of having to handle two kinds of configurations (`docker-compose` for dependencies and `aceptadora` for test subjects) so we decided to extend the functionality of `aceptadora` to completely replace the `docker-compose` and allow managing the lifecycle of all dependencies from the test.

You can read [the full story on Medium](https://medium.com/cabify-product/acceptance-testing-go-services-using-aceptadora-428254c34d56).

# Decisions

Everything in aceptadora accepts `t *testing.T` and everything does `require.NoError(t, err)` because in tests nobody's going to handle the errors anyway, so we apply a fail-fast strategy, removing the retured errors and keeping the API clean for clearer acceptance tests.

# Running

In order to handle multiple environments there are some stages in config loading. Notice that all the configs loaded expand the env vars set by `${VAR}` to their values from what's already loaded.

First, we load some very basic env-dependant config, deciding on env vars to load a local or gitlab config. 
This configuration mostly provides details about networking setup:
- Where can acceptance-tester reach the services? 
  Usually this would be `localhost`, but on Gitlab it's `docker` as we're running `dind`.
- Where can services reach the acceptance-tester? This will be set to the first local non-loopback IP address in the environment variable called `TESTER_ADDRESS`.
  You can set this variable to something more specific before running the test too, in which case it won't be overwritten. 

One may wonder: why don't we just decide all of that in some kind of test-loading shellscript? 
The answer is that deciding that from the test itself allows us running the tests from any IDE as a normal test, instead of having a proxy script. 
Of course loading some env vars would work, but since `aceptadora` can do this for you, why should we care?
This way we can make sure that test running is portable and doesn't require any external dependencies.

Then we load more env configs for the test itself, usually `acceptance.env`, which tells the acceptance test where the `aceptadora.yml` file is located, and how images from different docker registries are pulled.
Notice that `acceptance.env` can be specific to each suite you may have if their paths are different. 
You can load as many common configs as you want, loading an `acceptance.env` and then `../config/acceptance.env` for example.

Here `aceptadora.New()` would load the `aceptadora.yml` file from the provided dir and filename, and it will also set `${YAMLDIR}` variable for the yamls itself to be able to reference its configs and binds from where the yaml is located instead of from where the test is located: this allows us having multiple test suites in different folders using the same `aceptadora.yml` file.

Finally, we run services by just running `aceptadora.Run(ctx, "svc-name-in-the-yaml")`.

Aceptadora will also take care of stopping the services, you can call `aceptadora.Stop(ctx, svcName)` to stop one of them, or `StopAll(ctx)` to stop all the (still running) services.

# Unit tests

This package doesn't have unit tests. 
All the testing is performed by the example itself in the acceptance tests folder. 
The unit tests would require either defining interfaces for the functionality that docker provides and mocking them, which would overcomplicate the code without offering enough value in exchange, or testing using the docker real docker API, which is already covered by the acceptance tests.
However, this is opinionated. 
Feel free to disagree, open us an issue with your proposal, or even better, a pull request.
