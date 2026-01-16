# Changelog
All notable changes to this project will be documented in this file. See [conventional commits](https://www.conventionalcommits.org/) for commit guidelines.

- - -
## [v0.1.1](https://github.com/brad-jones/terraform-provider-denobridge/compare/de2faf30ffc938f870e0854e2f202e010265b40c..v0.1.1) - 2026-01-16
#### Bug Fixes
- (**resource**) while the state is optional, we don't require state to be returned from the deno script, setting this value here caused tfdocs to treat it as an input property which it is not. - ([de2faf3](https://github.com/brad-jones/terraform-provider-denobridge/commit/de2faf30ffc938f870e0854e2f202e010265b40c)) - [@brad-jones](https://github.com/brad-jones)
#### Documentation
- added tfplugindocs and generated thew docs folder - ([1eb2d92](https://github.com/brad-jones/terraform-provider-denobridge/commit/1eb2d922e4716356da8e8ce8ed1cfc57d5e9cddb)) - [@brad-jones](https://github.com/brad-jones)

- - -

## [v0.1.0](https://github.com/brad-jones/terraform-provider-denobridge/compare/a51f139d4a36aef7be24ba0594c59757dce477cc..v0.1.0) - 2026-01-15
#### Features
- added goreleaser and cocogitto config - ([f418c35](https://github.com/brad-jones/terraform-provider-denobridge/commit/f418c350d6145de9a3742084fe4a85440c86f751)) - [@brad-jones](https://github.com/brad-jones)
#### Bug Fixes
- (**downloader**) all assets are zip files not tar.gz - ([8b151d6](https://github.com/brad-jones/terraform-provider-denobridge/commit/8b151d6003c3d21caffdc3080c78f3f5a0c871fd)) - [@brad-jones](https://github.com/brad-jones)
- set CGO_ENABLED=0 for lint task to match build configuration (#1) - ([4e7c68f](https://github.com/brad-jones/terraform-provider-denobridge/commit/4e7c68f3d4cb60de0a506a253325e85b2633f36d)) - Copilot
#### Build system
- (**example**) should hopefully work across platforms now - ([78ba0ba](https://github.com/brad-jones/terraform-provider-denobridge/commit/78ba0bab009e533547901548d8e82d06762d2259)) - [@brad-jones](https://github.com/brad-jones)
- (**taskfile**) ensure CGO_ENABLED=0 for all tasks - ([8746ec2](https://github.com/brad-jones/terraform-provider-denobridge/commit/8746ec2beff9452dd1dbebb505d66b22ca78ed60)) - [@brad-jones](https://github.com/brad-jones)
- need to create the bin dir for goreleaser - ([61f160b](https://github.com/brad-jones/terraform-provider-denobridge/commit/61f160baeec26ff5fb9abbaeba291b00903aaba6)) - [@brad-jones](https://github.com/brad-jones)
#### Continuous Integration
- (**test**) provide a token so we do not get rate limited when downloading deno - ([9e03955](https://github.com/brad-jones/terraform-provider-denobridge/commit/9e039555c7528e2c27412630ac6e264722078ce3)) - [@brad-jones](https://github.com/brad-jones)
- enable release - ([62d310d](https://github.com/brad-jones/terraform-provider-denobridge/commit/62d310d9962568e6a12ef1c7f4011c517d82324b)) - [@brad-jones](https://github.com/brad-jones)
- disable release for now, still need to do some renaming - ([199cc25](https://github.com/brad-jones/terraform-provider-denobridge/commit/199cc25287d7425e8566cc758f9ca9c6d110ed1e)) - [@brad-jones](https://github.com/brad-jones)
- add build and tests for all platforms - ([e8198f9](https://github.com/brad-jones/terraform-provider-denobridge/commit/e8198f9805b103258cbfa08f225f897baf9eb89f)) - [@brad-jones](https://github.com/brad-jones)
#### Miscellaneous Chores
- (**example**) ignore the tf state - ([316abd4](https://github.com/brad-jones/terraform-provider-denobridge/commit/316abd4308f094d292c374747ff4010f21865bb2)) - [@brad-jones](https://github.com/brad-jones)
- update registry address - ([c35395c](https://github.com/brad-jones/terraform-provider-denobridge/commit/c35395cad26922da74a063ddbdcddce8f0019c47)) - [@brad-jones](https://github.com/brad-jones)
- rename to terraform-provider-denobridge - ([ad3bafb](https://github.com/brad-jones/terraform-provider-denobridge/commit/ad3bafb4762aee0a91fa77e81fb445a6db404947)) - [@brad-jones](https://github.com/brad-jones)
- make sure goreleaser can inject the version number - ([4339622](https://github.com/brad-jones/terraform-provider-denobridge/commit/4339622602fa9a2997e7d717b7dd964ebe7f421d)) - [@brad-jones](https://github.com/brad-jones)
- update go mod path - ([89055a1](https://github.com/brad-jones/terraform-provider-denobridge/commit/89055a17e31299c7dc14e1279ab4edf67b6d18f1)) - [@brad-jones](https://github.com/brad-jones)
- add the registry manifest file - ([0806324](https://github.com/brad-jones/terraform-provider-denobridge/commit/080632480b469e3052b18461a7aa0d5bec3aca81)) - [@brad-jones](https://github.com/brad-jones)
- add linting to lefthook - ([59ff9e4](https://github.com/brad-jones/terraform-provider-denobridge/commit/59ff9e464ef4665cf7e4ea7dce32128dcf43aa0f)) - [@brad-jones](https://github.com/brad-jones)
- we had a last minute change of heart and decided to just name this deno-tf-bridge instead of using tofu - ([d0b518d](https://github.com/brad-jones/terraform-provider-denobridge/commit/d0b518d1129f210d90d5d345bccaab8d640155f2)) - [@brad-jones](https://github.com/brad-jones)
- initial commit - ([a51f139](https://github.com/brad-jones/terraform-provider-denobridge/commit/a51f139d4a36aef7be24ba0594c59757dce477cc)) - [@brad-jones](https://github.com/brad-jones)
#### Style
- added golangci-lint and fixed all issues - ([cea8e44](https://github.com/brad-jones/terraform-provider-denobridge/commit/cea8e44f7e2f6f01878c65acb8e87821f0c4fdab)) - [@brad-jones](https://github.com/brad-jones)

- - -

Changelog generated by [cocogitto](https://github.com/cocogitto/cocogitto).