# Changelog
All notable changes to this project will be documented in this file. See [conventional commits](https://www.conventionalcommits.org/) for commit guidelines.

- - -
## [v0.4.1](https://github.com/brad-jones/terraform-provider-denobridge/compare/ec13ad526a522a9a06e3a19e844c90cf97f27d5f..v0.4.1) - 2026-02-15
#### Bug Fixes
- (**lib**) defensively check that a value is an object before attempting to delete properties from it - ([ec13ad5](https://github.com/brad-jones/terraform-provider-denobridge/commit/ec13ad526a522a9a06e3a19e844c90cf97f27d5f)) - [@brad-jones](https://github.com/brad-jones)

- - -

## [v0.4.0](https://github.com/brad-jones/terraform-provider-denobridge/compare/9b8ff4d5a036ebbf8a936b251878fe3a09dba922..v0.4.0) - 2026-02-14
#### Features
- (**resource**) add support for write only props - ([9b8ff4d](https://github.com/brad-jones/terraform-provider-denobridge/commit/9b8ff4d5a036ebbf8a936b251878fe3a09dba922)) - [@brad-jones](https://github.com/brad-jones)

- - -

## [v0.3.0](https://github.com/brad-jones/terraform-provider-denobridge/compare/6456dec71c41a5fe8317a1add2ac56c7508d3e75..v0.3.0) - 2026-02-13
#### Features
- (**datasource**) add support for sensitive state management - ([2253e14](https://github.com/brad-jones/terraform-provider-denobridge/commit/2253e14ba992acfe8fddaa4e9354302a7bd38f19)) - [@brad-jones](https://github.com/brad-jones)
- (**ephemeral**) add support for sensitive state management - ([1fcf2d4](https://github.com/brad-jones/terraform-provider-denobridge/commit/1fcf2d483ac84c49440820aa44382ac590de850e)) - [@brad-jones](https://github.com/brad-jones)
- (**lib**) inject DENOBRIDGE_VERSION constant for use in cdkts - ([b8d2792](https://github.com/brad-jones/terraform-provider-denobridge/commit/b8d2792d5c437a2a6f09bc5b3fb8cc5009663a9f)) - [@brad-jones](https://github.com/brad-jones)
- (**resource**) add support for sensitive state management - ([5112078](https://github.com/brad-jones/terraform-provider-denobridge/commit/5112078e7be165f54a4c684a6e41b460cf9071e5)) - [@brad-jones](https://github.com/brad-jones)
#### Bug Fixes
- (**action**) ensure CLI output alignment in InvokeProgress method - ([430f3f2](https://github.com/brad-jones/terraform-provider-denobridge/commit/430f3f2197a14810210a5c2f7649bf1113b96ff8)) - [@brad-jones](https://github.com/brad-jones)
- (**resource**) type error in example - ([548f15e](https://github.com/brad-jones/terraform-provider-denobridge/commit/548f15e04f9a737a98141bd40599cd608dabafa0)) - [@brad-jones](https://github.com/brad-jones)
- (**resource**) state is now truly optional - ([4561a66](https://github.com/brad-jones/terraform-provider-denobridge/commit/4561a661d47ef306ae568542467c79c5ca8c2f0d)) - [@brad-jones](https://github.com/brad-jones)
#### Documentation
- re-gen from templates - ([6f9b02b](https://github.com/brad-jones/terraform-provider-denobridge/commit/6f9b02b010309f794ab13dd7aa0d8cd0cde0c60e)) - [@brad-jones](https://github.com/brad-jones)
#### Tests
- (**resource**) improve comments and null safety in modifyPlan method - ([b3aefc2](https://github.com/brad-jones/terraform-provider-denobridge/commit/b3aefc27cf15c7b9a199ee7abfc40e098731fade)) - [@brad-jones](https://github.com/brad-jones)
#### Build system
- fix up golangci-lint install - ([637a4e8](https://github.com/brad-jones/terraform-provider-denobridge/commit/637a4e8c9549080ba1742099c0583f57a33c9a91)) - [@brad-jones](https://github.com/brad-jones)
- always format after generating docs - ([a7d50f1](https://github.com/brad-jones/terraform-provider-denobridge/commit/a7d50f1e9dd9487f76001927b5e34b4a89b0f191)) - [@brad-jones](https://github.com/brad-jones)
#### Miscellaneous Chores
- tweaks to examples - ([c643ed6](https://github.com/brad-jones/terraform-provider-denobridge/commit/c643ed63698d63e462ac63ea3f554423e3b9690c)) - [@brad-jones](https://github.com/brad-jones)
- bump pixi deps - ([6456dec](https://github.com/brad-jones/terraform-provider-denobridge/commit/6456dec71c41a5fe8317a1add2ac56c7508d3e75)) - [@brad-jones](https://github.com/brad-jones)
#### Style
- (**lib**) more whitespace improvements in comments for readability - ([9c368cf](https://github.com/brad-jones/terraform-provider-denobridge/commit/9c368cf89f6810cbcf6cbd0a231644e1ec7eab82)) - [@brad-jones](https://github.com/brad-jones)
- (**resource**) add spacing for improved readability in ResourceProviderMethods type - ([d169ed7](https://github.com/brad-jones/terraform-provider-denobridge/commit/d169ed79111d5dd10d1f76fa00fa1437be51cbde)) - [@brad-jones](https://github.com/brad-jones)

- - -

## [v0.2.7](https://github.com/brad-jones/terraform-provider-denobridge/compare/72f2c0e6fa1ab0f4faa6726a528b47684baa104c..v0.2.7) - 2026-02-11
#### Bug Fixes
- enhance FromDynamic and FromValue to support Tuple types - ([72f2c0e](https://github.com/brad-jones/terraform-provider-denobridge/commit/72f2c0e6fa1ab0f4faa6726a528b47684baa104c)) - [@brad-jones](https://github.com/brad-jones)
#### Build system
- stop trying to format the deno json file - ([11ba409](https://github.com/brad-jones/terraform-provider-denobridge/commit/11ba409ceffb2cb6aa61ecb88959076c4c6a68a8)) - [@brad-jones](https://github.com/brad-jones)

- - -

## [v0.2.6](https://github.com/brad-jones/terraform-provider-denobridge/compare/c2ed20674b74f03373c380d8d3576994c3538294..v0.2.6) - 2026-02-04
#### Bug Fixes
- all methods can return structured diagnostics now - ([75bad15](https://github.com/brad-jones/terraform-provider-denobridge/commit/75bad15fbfa4a469a559f5765a20d422c60b86f6)) - github-actions[bot]
#### Documentation
- updated docs to reflect that all methods can return structured diagnostics now - ([8aebf29](https://github.com/brad-jones/terraform-provider-denobridge/commit/8aebf298d9dc85ef723690406a6f02dfbf15da93)) - github-actions[bot]
#### Miscellaneous Chores
- bump pixi lock file - ([c2ed206](https://github.com/brad-jones/terraform-provider-denobridge/commit/c2ed20674b74f03373c380d8d3576994c3538294)) - github-actions[bot]

- - -

## [v0.2.5](https://github.com/brad-jones/terraform-provider-denobridge/compare/98bc4a27a4042aca618202debd6f3ac8be37c95c..v0.2.5) - 2026-02-03
#### Bug Fixes
- (**BaseJsonRpcProvider**) wrap all methods and log uncaught errors to stderr - ([454d80c](https://github.com/brad-jones/terraform-provider-denobridge/commit/454d80c4505db7bac048f9b5c13a3c66bdbb93c2)) - github-actions[bot]
- (**JSocket**) use a promise chain for all write operations - ([f62a889](https://github.com/brad-jones/terraform-provider-denobridge/commit/f62a889bce44661a203a053046c9607de94c1c88)) - github-actions[bot]
#### Documentation
- small change to opening line in readme - ([c366f33](https://github.com/brad-jones/terraform-provider-denobridge/commit/c366f33caff141dd8f3eb2a6ed154e406b61b6a6)) - github-actions[bot]
- updated the main readme to reflect the JSON-RPC changes - ([8d3acd7](https://github.com/brad-jones/terraform-provider-denobridge/commit/8d3acd7e23ce7df3b09cf9f270465389a1f7e27f)) - github-actions[bot]
- updated everything reflect the new JSON-RPC transport layer - ([c2709e3](https://github.com/brad-jones/terraform-provider-denobridge/commit/c2709e3ae1db7c1ace8b4949dd8f23f99d20d894)) - github-actions[bot]
#### Refactoring
- changed transport layer from HTTP to JSON-RPC - ([fe4dcc9](https://github.com/brad-jones/terraform-provider-denobridge/commit/fe4dcc996156a5892364782e82cbaf82b2c4907c)) - github-actions[bot]
#### Miscellaneous Chores
- bump pixi lock file - ([98bc4a2](https://github.com/brad-jones/terraform-provider-denobridge/commit/98bc4a27a4042aca618202debd6f3ac8be37c95c)) - github-actions[bot]

- - -

## [v0.2.4](https://github.com/brad-jones/terraform-provider-denobridge/compare/7609c9b8e515150e43776d377154a58acff362f9..v0.2.4) - 2026-01-28
#### Bug Fixes
- cater for file and other remote script urls when building the deno serve command - ([7609c9b](https://github.com/brad-jones/terraform-provider-denobridge/commit/7609c9b8e515150e43776d377154a58acff362f9)) - github-actions[bot]

- - -

## [v0.2.3](https://github.com/brad-jones/terraform-provider-denobridge/compare/aa6849c0899d246bd7e9ae94f56b37fec1e19167..v0.2.3) - 2026-01-28
#### Miscellaneous Chores
- log the final deno serve command that is executed by the provider - ([aa6849c](https://github.com/brad-jones/terraform-provider-denobridge/commit/aa6849c0899d246bd7e9ae94f56b37fec1e19167)) - github-actions[bot]

- - -

## [v0.2.2](https://github.com/brad-jones/terraform-provider-denobridge/compare/c4c8959eb918ef7a0cccc1914bf8e1f15104dcab..v0.2.2) - 2026-01-28
#### Bug Fixes
- (**build**) goreleasers changelog feature needs to be enabled for the --release-notes flag to work I think - ([c4c8959](https://github.com/brad-jones/terraform-provider-denobridge/commit/c4c8959eb918ef7a0cccc1914bf8e1f15104dcab)) - github-actions[bot]

- - -

## [v0.2.1](https://github.com/brad-jones/terraform-provider-denobridge/compare/5d8fd11ad21e78b551a108d6a57b04a01090b008..v0.2.1) - 2026-01-28
#### Bug Fixes
- (**build**) should get some nice changelog text in the github release now - ([5d8fd11](https://github.com/brad-jones/terraform-provider-denobridge/commit/5d8fd11ad21e78b551a108d6a57b04a01090b008)) - github-actions[bot]

- - -

## [v0.2.0](https://github.com/brad-jones/terraform-provider-denobridge/compare/a83aab7283a2ccded7342d3102059e82a282bb80..v0.2.0) - 2026-01-27
#### Features
- added config_file attribute and automatic config file discovery - ([3566c73](https://github.com/brad-jones/terraform-provider-denobridge/commit/3566c733e29e5519b2b273bc6887e981aea7a331)) - github-actions[bot]
#### Miscellaneous Chores
- bump pixi lock file - ([a83aab7](https://github.com/brad-jones/terraform-provider-denobridge/commit/a83aab7283a2ccded7342d3102059e82a282bb80)) - github-actions[bot]

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