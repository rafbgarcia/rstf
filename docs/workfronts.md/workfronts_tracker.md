# Workfront Tracker

Recommended implementation order and why each is sequenced there.

Note: Production-hardening requirements for SSR/runtime operations are explicitly tracked in Workfronts 01, 03, 08, and 09.

1. [01_request_model_actions](/Users/rafa/github.com/rafbgarcia/rstf/docs/workfronts/01_request_model_actions.md)
Reason: Defines the core runtime contract for everything beyond read-only SSR.

2. [02_validation_and_forms](/Users/rafa/github.com/rafbgarcia/rstf/docs/workfronts/02_validation_and_forms.md)
Reason: Immediately required once mutations/actions exist.

3. [03_error_model_and_resilience](/Users/rafa/github.com/rafbgarcia/rstf/docs/workfronts/03_error_model_and_resilience.md)
Reason: Stabilizes cross-cutting behavior before layering security and data concerns.

4. [04_auth_sessions_csrf](/Users/rafa/github.com/rafbgarcia/rstf/docs/workfronts/04_auth_sessions_csrf.md)
Reason: Security primitives depend on stable request, validation, and error semantics.

5. [05_data_layer_and_migrations](/Users/rafa/github.com/rafbgarcia/rstf/docs/workfronts/05_data_layer_and_migrations.md)
Reason: Solidifies persistent-state workflows after auth/session contracts are clear.

6. [06_config_and_environment_model](/Users/rafa/github.com/rafbgarcia/rstf/docs/workfronts/06_config_and_environment_model.md)
Reason: Formal config model should be locked before introducing cache/queue backends.

7. [07_caching_and_revalidation](/Users/rafa/github.com/rafbgarcia/rstf/docs/workfronts/07_caching_and_revalidation.md)
Reason: Caching strategy is safer once request, data, and config contracts are established.

8. [08_assets_and_production_build](/Users/rafa/github.com/rafbgarcia/rstf/docs/workfronts/08_assets_and_production_build.md)
Reason: Production artifact pipeline can now align with stable runtime behavior.

9. [09_observability_and_operations](/Users/rafa/github.com/rafbgarcia/rstf/docs/workfronts/09_observability_and_operations.md)
Reason: Telemetry should instrument finalized request/data/runtime behaviors.

10. [10_background_jobs_and_scheduling](/Users/rafa/github.com/rafbgarcia/rstf/docs/workfronts/10_background_jobs_and_scheduling.md)
Reason: Async system design benefits from mature config, observability, and data contracts.
