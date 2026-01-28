# Changelog

## [0.28.0](https://github.com/vitalvas/gokit/compare/v0.27.0...v0.28.0) (2026-01-21)


### Features

* **shamir:** add Shamir's Secret Sharing implementation ([2f072cd](https://github.com/vitalvas/gokit/commit/2f072cd0a7691383145d162a9699025291df33f3))


### Bug Fixes

* **xlogger:** auto-detect source path from build info ([23cf878](https://github.com/vitalvas/gokit/commit/23cf8788608263b1b9e4f9fc532cde05a6a93c11))

## [0.27.0](https://github.com/vitalvas/gokit/compare/v0.26.0...v0.27.0) (2026-01-20)


### Features

* **wirefilter:** add cidr() and cidr6() functions for IP masking ([b7d1716](https://github.com/vitalvas/gokit/commit/b7d17169bcee75b887ca20716048d32816807e80))
* **wirefilter:** add function control and improve test coverage ([9393082](https://github.com/vitalvas/gokit/commit/93930820f64838d70a61159bb187712225d5899f))
* **wirefilter:** add missing operators and aliases ([a980bd3](https://github.com/vitalvas/gokit/commit/a980bd3888e6f929418ce324b1936bcb263cbcfc))
* **wirefilter:** add missing operators and Cloudflare compatibility tests ([9d7cbdc](https://github.com/vitalvas/gokit/commit/9d7cbdc142e4a09dc29f10aa2ea00bcbb64d6844))
* **wirefilter:** add raw strings, array index, array unpack, and custom lists ([830a984](https://github.com/vitalvas/gokit/commit/830a984b239fc62964fbf1e4ca109f1828c29bce))
* **wirefilter:** add transformation functions ([8919979](https://github.com/vitalvas/gokit/commit/8919979bc5485ed6e8c4f17e928c5aeca8e74979))


### Bug Fixes

* **wirefilter:** address correctness and safety issues from review ([e8e7d14](https://github.com/vitalvas/gokit/commit/e8e7d140454799bb233adc5f367acece57989777))
* **xcmd:** prevent test timeout in PeriodicRunWithSignal ([e12681d](https://github.com/vitalvas/gokit/commit/e12681d517983c20a13d2e626c54042561d57fbf))

## [0.26.0](https://github.com/vitalvas/gokit/compare/v0.25.0...v0.26.0) (2026-01-18)


### Features

* **wirefilter:** add map type support and field-to-field comparisons ([59a5c14](https://github.com/vitalvas/gokit/commit/59a5c14d364a5513b469ab7b1bb832ba5466d83f))

## [0.25.0](https://github.com/vitalvas/gokit/compare/v0.24.0...v0.25.0) (2026-01-18)


### Features

* **wirefilter:** add field presence checking and array-to-array operations ([6b36711](https://github.com/vitalvas/gokit/commit/6b3671111fd82226321dccee96743cfd5b8ebb5f))

## [0.24.0](https://github.com/vitalvas/gokit/compare/v0.23.1...v0.24.0) (2026-01-14)


### Features

* **xconfig:** add WithStrict option for strict JSON/YAML parsing ([03164af](https://github.com/vitalvas/gokit/commit/03164af05052211b1af935c48506e8a48ea70992))

## [0.23.1](https://github.com/vitalvas/gokit/compare/v0.23.0...v0.23.1) (2026-01-14)


### Bug Fixes

* **xconfig:** strip quotes from dotenv values ([38ec71d](https://github.com/vitalvas/gokit/commit/38ec71d14d2222fa949b48c34a0d7151a1bfb401))

## [0.23.0](https://github.com/vitalvas/gokit/compare/v0.22.0...v0.23.0) (2026-01-14)


### Features

* **xconfig:** add defaults for WithDotenv and WithEnvMacroRegex ([4103d37](https://github.com/vitalvas/gokit/commit/4103d374cc7b8ba0170e825c694ecdc7292f7410))


### Bug Fixes

* **wirefilter:** move regex and CIDR caches to Filter instance ([c6a31b2](https://github.com/vitalvas/gokit/commit/c6a31b26d87685afe987adcc8da09c8edf760ac7))

## [0.22.0](https://github.com/vitalvas/gokit/compare/v0.21.0...v0.22.0) (2026-01-11)


### Features

* **fastcdc:** create fastcdc ([3dba0eb](https://github.com/vitalvas/gokit/commit/3dba0eb5fc58c9b48278ca3b7549e66647625c2d))
* **xnet:** add PROXY protocol v1/v2 listener wrapper ([2107db7](https://github.com/vitalvas/gokit/commit/2107db7d5276bb51729d317f313ae9008f14440a))


### Bug Fixes

* fix prealloc linter warnings ([fab1905](https://github.com/vitalvas/gokit/commit/fab1905a4fa22ba6d5f54ec075856a939afbb0cf))
* resolve linter warnings in fuzz tests ([bf89693](https://github.com/vitalvas/gokit/commit/bf8969356fcdfe4f5671fa9c42193a9fe19d221e))
* **xnet:** fix prealloc linter warnings in tests ([f932658](https://github.com/vitalvas/gokit/commit/f932658b2128b54c6e4247f6a82f9f0a65749c61))


### Performance Improvements

* **wirefilter:** add benchmarks, fuzz tests and reduce allocations ([cb9f3c0](https://github.com/vitalvas/gokit/commit/cb9f3c0e379e72e206865fdda5577ca4e8641770))
* **xnet:** add benchmarks, fuzz tests and reduce allocations ([cf0fa3c](https://github.com/vitalvas/gokit/commit/cf0fa3c3c01034f2a0df2e05fd765cda62172511))

## [0.21.0](https://github.com/vitalvas/gokit/compare/v0.20.0...v0.21.0) (2025-11-23)


### Features

* create wirefilter ([cdc2629](https://github.com/vitalvas/gokit/commit/cdc2629453878bef3e37c238972419f0321c602a))
* **xcmd:** add errgroup ([cb71999](https://github.com/vitalvas/gokit/commit/cb7199918e0d8a3ed7579c45665a6168fbc4619e))

## [0.20.0](https://github.com/vitalvas/gokit/compare/v0.19.0...v0.20.0) (2025-10-30)


### Features

* add rates ([f4a611c](https://github.com/vitalvas/gokit/commit/f4a611c83508fb7c4269dd2e6310ab60a4c6d9bf))
* **countmin:** create spacesaving ([f581aed](https://github.com/vitalvas/gokit/commit/f581aedfeae1c4fd4d201f1b8cd6e289be3b7bba))
* **cuckoo:** create cuckoo ([b4cd992](https://github.com/vitalvas/gokit/commit/b4cd992519f407c9e49313d175ed9624577b75e5))
* **ewma:** create ewma ([d10c38d](https://github.com/vitalvas/gokit/commit/d10c38d7afdf4f4443067adb7aee7c133e4ab351))
* **hyperloglog:** create hyperloglog ([f4425ac](https://github.com/vitalvas/gokit/commit/f4425ac5f91f368a299e4fe662f6a6b49159245a))
* **markov:** full refactor lib ([51c8bc5](https://github.com/vitalvas/gokit/commit/51c8bc5e95cab3d5d746e410a0f8cd8ed88ec59a))
* **spacesaving:** create spacesaving ([5574643](https://github.com/vitalvas/gokit/commit/5574643aeed84fa662deed0456acf922f6a7a0dd))
* **tdigest:** create t-digest ([e62c814](https://github.com/vitalvas/gokit/commit/e62c814470c48af284c32ab9c28d3aa61673a464))
* **xconfig:** add envSeparator ([80e9e9b](https://github.com/vitalvas/gokit/commit/80e9e9b20b0809e345582af4bf4d20e537ff702b))
* **xconfig:** add support dotenv ([737690c](https://github.com/vitalvas/gokit/commit/737690c9eb19428fd495de72c39743a71ce04c2e))
* **xentropy:** add entropy ([f5bdb2f](https://github.com/vitalvas/gokit/commit/f5bdb2f2a10e23f97fb95b53b634e4ad779f3542))


### Bug Fixes

* linter errors ([769dcf9](https://github.com/vitalvas/gokit/commit/769dcf9522e7fcc07863649cd209b0c82d218a7c))
* linter errors ([ed155a1](https://github.com/vitalvas/gokit/commit/ed155a107b17521c00c346ff39e63376fd622b64))
* linter errors ([9549109](https://github.com/vitalvas/gokit/commit/95491091cb7e7fea96327573eb502d1d2246ee7b))
* **xconfig:** support slice objects ([af18fd3](https://github.com/vitalvas/gokit/commit/af18fd35f2148e92e0a17afce74ebd972dac4da6))

## [0.19.0](https://github.com/vitalvas/gokit/compare/v0.18.1...v0.19.0) (2025-10-28)


### Features

* **xnet:** add cidr_macher ([f765354](https://github.com/vitalvas/gokit/commit/f7653548cb0dce582feddc262bdea993db7a7318))
* **xnet:** add cidr_merge ([87faffd](https://github.com/vitalvas/gokit/commit/87faffd268712784ce7ec899b9fd75d16665e850))
* **xnet:** add cidr_split ([de5dcb1](https://github.com/vitalvas/gokit/commit/de5dcb1897be2d1122469006c802be05ceb91116))

## [0.18.1](https://github.com/vitalvas/gokit/compare/v0.18.0...v0.18.1) (2025-09-09)


### Bug Fixes

* load empty var ([3e04549](https://github.com/vitalvas/gokit/commit/3e04549aa7964be130d30a6bb0d00cf25aa4d20e))

## [0.18.0](https://github.com/vitalvas/gokit/compare/v0.17.1...v0.18.0) (2025-08-10)


### Features

* **xconfig:** add support default tag ([6e46bbc](https://github.com/vitalvas/gokit/commit/6e46bbcd34cb656b248c80e88ed7c143fc7f1c86))
* **xconfig:** add support env tag ([c990c6e](https://github.com/vitalvas/gokit/commit/c990c6eb06e3fafc0a58927fc80b84469003871b))
* **xconfig:** add support time.Duration ([36430f1](https://github.com/vitalvas/gokit/commit/36430f1bf80828480776ec3b5e0e0e49858d8e58))


### Bug Fixes

* **xconfig:** support time.Duration and refactor tests ([fc2bad2](https://github.com/vitalvas/gokit/commit/fc2bad2c799d45f94fc7c8ba2e99623bc91f4a63))

## [0.17.1](https://github.com/vitalvas/gokit/compare/v0.17.0...v0.17.1) (2025-07-26)


### Bug Fixes

* **xconfig:** merge keys by priority ([4e8890d](https://github.com/vitalvas/gokit/commit/4e8890d446fdaefa3120d2cf206c80b86cfa075f))

## [0.17.0](https://github.com/vitalvas/gokit/compare/v0.16.0...v0.17.0) (2025-07-12)


### Features

* add read dir ([4fd1490](https://github.com/vitalvas/gokit/commit/4fd1490e24ba7b73db44c6f94922c4b872b44aa2))
* add xconfig ([b400d07](https://github.com/vitalvas/gokit/commit/b400d079ea1bc982ffabbb6c7dab983154aca204))
* refactor bloomfilter ([c195a6e](https://github.com/vitalvas/gokit/commit/c195a6e0edafe4a7910750c989e022daccfc56c0))


### Bug Fixes

* camelCase format ([a162042](https://github.com/vitalvas/gokit/commit/a16204255d5217cca7132786b2507176afce863b))

## [0.16.0](https://github.com/vitalvas/gokit/compare/v0.15.0...v0.16.0) (2025-03-07)


### Features

* create bloomfilter ([024ef71](https://github.com/vitalvas/gokit/commit/024ef71d72023c8a284a5325d4565a0cfbb23e68))

## [0.15.0](https://github.com/vitalvas/gokit/compare/v0.14.0...v0.15.0) (2025-02-15)


### Features

* create xconvert ([6e6fbe6](https://github.com/vitalvas/gokit/commit/6e6fbe6913cde3b808015885e01f696ecffbec44))


### Bug Fixes

* tests for xconvert ([c77838f](https://github.com/vitalvas/gokit/commit/c77838f66b52c455c8b0e41179e41dc3ed1077fd))

## [0.14.0](https://github.com/vitalvas/gokit/compare/v0.13.1...v0.14.0) (2024-12-12)


### Features

* add replacemap ([07eebbb](https://github.com/vitalvas/gokit/commit/07eebbbd3146bee6b165abc59d6d53b81cae1277))


### Bug Fixes

* replacemap ([a720370](https://github.com/vitalvas/gokit/commit/a7203708340b4d09ce4883cf852a9fca84df8267))
* run test for sortmap ([29813f5](https://github.com/vitalvas/gokit/commit/29813f5f4c8aef5fb3ece57f4ae6094c7aee1538))

## [0.13.1](https://github.com/vitalvas/gokit/compare/v0.13.0...v0.13.1) (2024-11-25)


### Bug Fixes

* built-in functions ([bdcbe92](https://github.com/vitalvas/gokit/commit/bdcbe92aba8865ae295010bf2628a09c84e7bec0))

## [0.13.0](https://github.com/vitalvas/gokit/compare/v0.12.0...v0.13.0) (2024-09-30)


### Features

* add xlogger ([eecec7d](https://github.com/vitalvas/gokit/commit/eecec7d4790b7c53b74dc14da95e52bcc982644b))

## [0.12.0](https://github.com/vitalvas/gokit/compare/v0.11.0...v0.12.0) (2024-09-10)


### Features

* add xnet cidr contains ([93a9d9d](https://github.com/vitalvas/gokit/commit/93a9d9d57bc5e12fe27f8a33b6b5d37dc8abad8c))

## [0.11.0](https://github.com/vitalvas/gokit/compare/v0.10.0...v0.11.0) (2024-09-10)


### Features

* add xnet strip address ([1bab313](https://github.com/vitalvas/gokit/commit/1bab313fae2562b25e8cc12d42fabd792c21f3d1))

## [0.10.0](https://github.com/vitalvas/gokit/compare/v0.9.0...v0.10.0) (2024-08-24)


### Features

* create xstrings simple template ([d917b5a](https://github.com/vitalvas/gokit/commit/d917b5a8c58a2405eb74a3495a0e64b6623b4f01))

## [0.9.0](https://github.com/vitalvas/gokit/compare/v0.8.0...v0.9.0) (2024-08-23)


### Features

* add ci release-please ([e9172bb](https://github.com/vitalvas/gokit/commit/e9172bb473af00c32a39ed26774fd3f8a15d39dc))
* add ci release-please ([83d85e6](https://github.com/vitalvas/gokit/commit/83d85e6c044b0d1d664dbf4f2ccbc266525e6593))
* add tests for SortMap ([68fba7d](https://github.com/vitalvas/gokit/commit/68fba7d1157dc1be0a25de35ebfa173a89a3ab2f))
* move gomarkov ([c844fc7](https://github.com/vitalvas/gokit/commit/c844fc773ab051ac84aecb076e3cba816cc3fb64))
* move uxid ([65298b4](https://github.com/vitalvas/gokit/commit/65298b431fe0fc8aa399e7fc9b44bffe2652ab5d))
* update deps ([9a3571c](https://github.com/vitalvas/gokit/commit/9a3571c26299cae18119bf46c55107a330ce5754))


### Bug Fixes

* issue with release-please ([cfde00f](https://github.com/vitalvas/gokit/commit/cfde00fa4ade4bb331a6f66cac97fc2dc948dcda))
* issue with release-please ([c5c38e9](https://github.com/vitalvas/gokit/commit/c5c38e9b44e7913b4ac6f9158e3fd2cc5cf6063e))
* issue with release-please ([bdd5140](https://github.com/vitalvas/gokit/commit/bdd514000d18e0bf06d5fb4c42a001b8c019466d))
* issue with release-please ([14b80d6](https://github.com/vitalvas/gokit/commit/14b80d6f2bd2eccad6c7c36ede6457eb6dfb46bb))
* issue with release-please ([eae6c4c](https://github.com/vitalvas/gokit/commit/eae6c4c5ea22c5465e30cd419ef2c502d5a38b46))
* linter errors ([ef219c8](https://github.com/vitalvas/gokit/commit/ef219c86e52d2d7ffe4385d4f6adcc94a3c16067))
