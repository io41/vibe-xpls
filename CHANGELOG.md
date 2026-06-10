# Changelog

## [0.0.4](https://github.com/io41/vibe-xpls/compare/v0.0.3...v0.0.4) (2026-06-10)


### Features

* add generated schema coverage artifacts ([4841152](https://github.com/io41/vibe-xpls/commit/48411523f00b27dba7b62d69ccafd81033cc1f35))
* add release-aware schema model ([2dc643f](https://github.com/io41/vibe-xpls/commit/2dc643f1b10f82964e11937492d26d47d1805a73))
* add scheduled schema drift check ([6b83ccb](https://github.com/io41/vibe-xpls/commit/6b83ccb585099153ab320b7a18e4ec7de77ca4a1))
* add schema coverage baseline contract ([27a4f4a](https://github.com/io41/vibe-xpls/commit/27a4f4aebdd5ccf3eb9b7f70dda7e1e4dfe0e6c4))
* add schema coverage generator commands ([1ac64f0](https://github.com/io41/vibe-xpls/commit/1ac64f00a24303ec14d9b33fa05edb8b3de863e5))
* add schema generator ([651d07a](https://github.com/io41/vibe-xpls/commit/651d07a3804e424b2775a1f175101299d00c0bfb))
* add YAML document path value lookup ([c1f886a](https://github.com/io41/vibe-xpls/commit/c1f886a454da3838d4c228e8c3baba2800db228c))
* classify generated schema coverage ([c51d8ea](https://github.com/io41/vibe-xpls/commit/c51d8ea2035a8306a5336daab9867e64adcf16f8))
* complete from generated schema paths ([b911d86](https://github.com/io41/vibe-xpls/commit/b911d86af70193c7483c2a6b73edfc197177231e))
* cover function input completions over lsp ([f1f24da](https://github.com/io41/vibe-xpls/commit/f1f24dae62087854de2ddd61435df3156ae3aebd))
* dispatch function input completions ([61aeee0](https://github.com/io41/vibe-xpls/commit/61aeee0db5f4053747f3e0c465af1b714c00fdb2))
* enforce schema coverage baseline ([6c751d1](https://github.com/io41/vibe-xpls/commit/6c751d16edf04ee75abaea9fb4f5753d0c970c0b))
* extract schema coverage targets ([c2f8412](https://github.com/io41/vibe-xpls/commit/c2f8412b6e2fab584a0e54491cc036a2b14ca3ac))
* generate Crossplane core schemas ([a099e82](https://github.com/io41/vibe-xpls/commit/a099e82f889ae455f0d29a4f314ff637c599a5b5))
* improve schema coverage reporting ([9892819](https://github.com/io41/vibe-xpls/commit/9892819cb5037430c19910f59a3b04a67c25676d))
* load generated schema bundle ([2d8c24c](https://github.com/io41/vibe-xpls/commit/2d8c24c5a94e2ac7392b501d67b6a47923cbf594))
* load workspace provider crd schemas ([ccf3213](https://github.com/io41/vibe-xpls/commit/ccf3213fee2cce3715265fec792a79feb1e618e2))
* load workspace xrd schemas ([2de313b](https://github.com/io41/vibe-xpls/commit/2de313b10318a01e681952b0192360a58851beb7))
* render schema coverage artifacts ([cf82618](https://github.com/io41/vibe-xpls/commit/cf826183ba640d84f4ef2df799865bb0b4f6cfaa))
* report schema completion degradation ([e79c61b](https://github.com/io41/vibe-xpls/commit/e79c61ba4ca67fb37e9496b56d19c2176182da84))
* resolve schema release per package ([1bbb731](https://github.com/io41/vibe-xpls/commit/1bbb731df9ef45b4d79f486c029e901d0bf0b2b7))
* support direct function input completions ([c263ecc](https://github.com/io41/vibe-xpls/commit/c263ecc551952736993725f05d5a01ac25f68fd5))


### Bug Fixes

* align YAML spans with source positions ([a21efea](https://github.com/io41/vibe-xpls/commit/a21efea202afa9848dc3233957816e57d748184f))
* canonicalize workspace schema diagnostic uris ([ceff09d](https://github.com/io41/vibe-xpls/commit/ceff09d0e492c873ce4d8ccfdb445598215fe182))
* complete first keys in yaml array items ([9465279](https://github.com/io41/vibe-xpls/commit/9465279650d67cc473d6ca5fcabc43f21c5c5934))
* constrain completion fallback and version ranges ([896fbd1](https://github.com/io41/vibe-xpls/commit/896fbd135a72c0f44c93cce0fadff37fd7a9e55c))
* copy schema field metadata ([a836179](https://github.com/io41/vibe-xpls/commit/a836179918a9bdb3d3e3618c5b4e7315bf50888b))
* correct schema coverage report totals ([8db4cb4](https://github.com/io41/vibe-xpls/commit/8db4cb45e37badaf0b75d6ff035a2190823b500a))
* document compatibility schema parent completions ([d317511](https://github.com/io41/vibe-xpls/commit/d31751128b856548b4b4210f6952b6b06ca28516))
* gate direct function input completions on outer schema ([97db211](https://github.com/io41/vibe-xpls/commit/97db2114c4c61d1b1abe3a7a4be6a2431f0d9163))
* generate compatibility schema data ([bfe5611](https://github.com/io41/vibe-xpls/commit/bfe5611facce6f952051ae5853bd3ddda16f58c9))
* harden coverage target extraction ([cce1303](https://github.com/io41/vibe-xpls/commit/cce1303f9a4d4f5a671f0de37f1b4b1212082d03))
* harden schema generator paths and refs ([89c4dff](https://github.com/io41/vibe-xpls/commit/89c4dff1b448d3ccf8834f11a97b0570bec0f27d))
* harden workspace schema loading ([dbab7ca](https://github.com/io41/vibe-xpls/commit/dbab7ca262050e00d652e89e10870c11571802e4))
* honor package marker edits and workspace schemas ([a937006](https://github.com/io41/vibe-xpls/commit/a937006519b1df27caa07504a303a5abc82e4f71))
* ignore workspace schema sources in ignored dirs ([8353b51](https://github.com/io41/vibe-xpls/commit/8353b51798d29b06ecd30de78383cb3bec3260f7))
* improve schema coverage check diagnostics ([b27576e](https://github.com/io41/vibe-xpls/commit/b27576e4668b878c84cee5f924e26355a5a89ffb))
* preserve generated schema bundle behavior ([dbcc166](https://github.com/io41/vibe-xpls/commit/dbcc16621a142503d2f14ae552a77f8cd393c121))
* preserve generated schema compatibility ([dd17c47](https://github.com/io41/vibe-xpls/commit/dd17c477e99c60ecc28f35d7ae71865a2a908b9d))
* preserve hover at scalar boundaries ([9897f2c](https://github.com/io41/vibe-xpls/commit/9897f2c4c88116a58de5a631d01ff3f9087d026d))
* preserve int-or-string schema metadata ([9a080ae](https://github.com/io41/vibe-xpls/commit/9a080ae7f95fa8b711c15586c3c411671f23d076))
* publish workspace schema diagnostics for related sources ([ada1335](https://github.com/io41/vibe-xpls/commit/ada13351d070cc9e85558f7ebe23831294766842))
* refine schema coverage state reporting ([e85cb60](https://github.com/io41/vibe-xpls/commit/e85cb603bce92514425f5051b569dca330c80a26))
* render completion docs as markdown ([78151ec](https://github.com/io41/vibe-xpls/commit/78151ec0959de3872fa91b492c56e203e5c6a1b8))
* resolve annotated schema drift tags ([eb81b38](https://github.com/io41/vibe-xpls/commit/eb81b38749e68fd033614f38d3a8719322d31d19))
* scope workspace schemas by package ([c9dcfe9](https://github.com/io41/vibe-xpls/commit/c9dcfe9ca5d4c060ecc92e534f9fc1a86518f935))
* stabilize hover path selection ([36fb7bd](https://github.com/io41/vibe-xpls/commit/36fb7bdc0e500eb6d3f9e7dead0d7d0d04e00ccc))
* surface workspace schema refresh diagnostics ([a7c0999](https://github.com/io41/vibe-xpls/commit/a7c09991e81d4780068d1f1a6873692e74b07d92))
* throttle stable completion suppressions ([88854e9](https://github.com/io41/vibe-xpls/commit/88854e930d788565e032d7436d5bd4cf3eecb451))
* tighten completion degradation reporting ([af3357b](https://github.com/io41/vibe-xpls/commit/af3357b2a5392de6758e7b56100f7f1073b3eea5))
* tighten schema coverage baseline ratchet ([36071dc](https://github.com/io41/vibe-xpls/commit/36071dc6d499989629eb2ef49c3e7a3c016af26e))


### Documentation

* add project roadmap ([8127333](https://github.com/io41/vibe-xpls/commit/8127333380a479c68c6d274f161102d6cb020fe6))
* add schema coverage ratchet implementation plan ([1d42bf9](https://github.com/io41/vibe-xpls/commit/1d42bf9ad68b82e51900dcb8b9ea3d69a66ddbb6))
* align completion documentation specs ([075a9a9](https://github.com/io41/vibe-xpls/commit/075a9a93c597836c030bd54ee0812d22cf2b6f4c))
* clarify experimental project status ([42c6058](https://github.com/io41/vibe-xpls/commit/42c6058091d35a95443e5d5e3e03c100ed28838f))
* clarify function input dispatch gating ([6da0b4a](https://github.com/io41/vibe-xpls/commit/6da0b4afccb18fdd0cf7e477b9cdf7224f0ecd49))
* close function input schema dispatch ([2282cc0](https://github.com/io41/vibe-xpls/commit/2282cc017d2759d304d8457699ee8d7aa3aad8cb))
* design completion ux cleanup ([0589bd5](https://github.com/io41/vibe-xpls/commit/0589bd5ffa1ad850a3a4e4a529996e2e5dd6a0ff))
* design generated completion foundation ([1a8a440](https://github.com/io41/vibe-xpls/commit/1a8a44099fa59296dc0c9d486f884ffaef530644))
* design schema coverage ratchet ([2c6b6ab](https://github.com/io41/vibe-xpls/commit/2c6b6ab99060ed54fa36b3767362c1f3549fbe78))
* document generated schema bundle ([8c2c73e](https://github.com/io41/vibe-xpls/commit/8c2c73e24a0183987ecc27499ad5de03ee3128a9))
* plan completion ux cleanup ([15f736c](https://github.com/io41/vibe-xpls/commit/15f736ceb808afffc553ec046761c1ff714b4523))
* plan function input schema dispatch ([6f18996](https://github.com/io41/vibe-xpls/commit/6f18996d6fd476346e686373e52deeb7a286f351))
* plan generated completion foundation ([216568f](https://github.com/io41/vibe-xpls/commit/216568fcb2fb5d3e7f68c9e9c61b868b9c6583b9))
* refine schema coverage ratchet design ([806e184](https://github.com/io41/vibe-xpls/commit/806e184ab3b4541da8492afb676c2e42d078d9c2))
* track crossplane pin refresh ([1b3e7cd](https://github.com/io41/vibe-xpls/commit/1b3e7cdca9172f8546a54f8e7d1a520f8d1677a8))
* update workspace schema roadmap status ([43846d8](https://github.com/io41/vibe-xpls/commit/43846d814e4adccd5743f019ce1af2031c6ce92c))


### Tests

* cover array item completion edits ([ddafa95](https://github.com/io41/vibe-xpls/commit/ddafa9560d3e5bc0aac8ffe90e198ed2b7da1403))
* cover empty completion documentation ([3c1231c](https://github.com/io41/vibe-xpls/commit/3c1231caf9a0909440ad185d3bf7dfaba3351c5e))
* cover nested workspace schema boundaries ([a25b77c](https://github.com/io41/vibe-xpls/commit/a25b77cd98176963db888014913bcbec1f5ead03))
* cover schema generator path guards ([821633f](https://github.com/io41/vibe-xpls/commit/821633f7bc89e7245744633565f96d66c3b7a5de))
* cover workspace schema conflicts ([c4e53d7](https://github.com/io41/vibe-xpls/commit/c4e53d7f644d12f5bcc2df4f699a8bca0566dcad))
* harden completion metadata assertions ([4504e9a](https://github.com/io41/vibe-xpls/commit/4504e9a163573bd96dda07bea978b77a6e8225dc))

## [0.0.3](https://github.com/io41/vibe-xpls/compare/v0.0.2...v0.0.3) (2026-05-20)


### Bug Fixes

* add completion presentation metadata ([6e1bacb](https://github.com/io41/vibe-xpls/commit/6e1bacbff428247006856d889928004219e4332e))
* omit generic completion detail ([db31793](https://github.com/io41/vibe-xpls/commit/db3179309130d6f2cf63adfec9867a2848f39b15))
* suppress existing fallback completions ([c08614a](https://github.com/io41/vibe-xpls/commit/c08614a2fcdaf4f38b53dc3ddf07650048603ec2))


### Documentation

* design completion presentation slice ([12c355d](https://github.com/io41/vibe-xpls/commit/12c355d6e0f7f8cc50ddcfbd6b56196772fa18dd))
* design completion presentation slice ([74b43be](https://github.com/io41/vibe-xpls/commit/74b43be68173a2c0976da4c0f689ea85c8abd8db))
* investigate completion fallback bug ([c7109d0](https://github.com/io41/vibe-xpls/commit/c7109d08704391cf608afbe35a79ac21c18e5836))
* mark completion presentation plan complete ([24e7387](https://github.com/io41/vibe-xpls/commit/24e7387719368ebae9b61b69b2843bd8c78fb8f6))
* plan completion presentation slice ([54caf1a](https://github.com/io41/vibe-xpls/commit/54caf1a4fef9fddf2002d106afc48ae1e320654b))
* remove obsolete zed integration history ([84cede7](https://github.com/io41/vibe-xpls/commit/84cede7fece6dc93d9141defe347aad7790318f1))
* remove resolved follow-ups ([782793f](https://github.com/io41/vibe-xpls/commit/782793f7df9f0ba79fa05d3a29806e5517ca7c08))
* track zed completion follow-ups ([d3f3d2a](https://github.com/io41/vibe-xpls/commit/d3f3d2a2f87ab83cd272e0e5c6953e0d600a740d))
* update zed integration instructions ([da9df84](https://github.com/io41/vibe-xpls/commit/da9df84f259fab47ae3e57c52ba5f709dd20b928))


### Tests

* add completion presentation metadata coverage ([57c047a](https://github.com/io41/vibe-xpls/commit/57c047a1d2bdd406fe901bf3890e1ad8eb7c6861))

## [0.0.2](https://github.com/io41/vibe-xpls/compare/v0.0.1...v0.0.2) (2026-05-20)


### Bug Fixes

* preserve completion edit indentation in zed ([bd65d07](https://github.com/io41/vibe-xpls/commit/bd65d0745f6b8a58ffc5876feec79ed539be70ab))

## 0.0.1 (2026-05-19)


### Features

* add analyzer path safety limits ([fc4f51d](https://github.com/io41/vibe-xpls/commit/fc4f51d577151cd128b45a26afe24a005bfe1d89))
* add internal analyzer debug cli ([80ab646](https://github.com/io41/vibe-xpls/commit/80ab646ef6c5f9f5ea86280ece581a883a139ced))
* add lsp jsonrpc framing ([3c080fc](https://github.com/io41/vibe-xpls/commit/3c080fc9253ed8b657fa18a1bd3f6f1d265b12d9))
* add schema index with builtins ([ff9a165](https://github.com/io41/vibe-xpls/commit/ff9a1658ea24e512ec7d012b69da1e136fb15a2b))
* add source position conversion ([62aab08](https://github.com/io41/vibe-xpls/commit/62aab085eb061e758da057df9262f886d7d1f647))
* detect crossplane workspace packages ([c4b97a2](https://github.com/io41/vibe-xpls/commit/c4b97a28e49c64b3a0047bbd28f7209741c4c859))
* expose analyzer diagnostics hover completion ([75025c9](https://github.com/io41/vibe-xpls/commit/75025c961879b34df2c954fc89f092b98f37e7d2))
* parse mixed yaml template documents ([2f85c44](https://github.com/io41/vibe-xpls/commit/2f85c44a5243be4a9ddad9cbcb5903c577359987))
* scaffold vibe xpls binary ([032c1be](https://github.com/io41/vibe-xpls/commit/032c1be3c7b28469f4689bd13bf8a7bc0b876a6f))
* serve analyzer over lsp ([ce36a62](https://github.com/io41/vibe-xpls/commit/ce36a626f8adecc7916010fc4e7b15018b5c50a2))
* track analyzer document generations ([3850ac8](https://github.com/io41/vibe-xpls/commit/3850ac8ebc6790b9ad0d37f901ac76e0425389fc))


### Bug Fixes

* add completion text edits ([5dbb591](https://github.com/io41/vibe-xpls/commit/5dbb5910213237749434ec3b6fa484648e01254a))
* allow shared analyzer root context ([ab156a2](https://github.com/io41/vibe-xpls/commit/ab156a289b57ae6c08a453ef05c2e99d2b198381))
* bound range-derived yaml stability ([e02cfb4](https://github.com/io41/vibe-xpls/commit/e02cfb43f36998db817c57f90947acc2cb90365e))
* constrain completion to yaml key contexts ([2b06ed6](https://github.com/io41/vibe-xpls/commit/2b06ed6e102202e96a79034f36b255174708240d))
* dedent root completion edits ([76abbb0](https://github.com/io41/vibe-xpls/commit/76abbb0cc7c7c8af82c5a643c4844b0f5a269ae8))
* deduplicate workspace package roots ([4c2378c](https://github.com/io41/vibe-xpls/commit/4c2378c1a9a32021ec0afe89c789c4734dfaf1a5))
* disambiguate analyzer schema roots ([25f27ca](https://github.com/io41/vibe-xpls/commit/25f27ca998de5530c46a96765fb1312b04201baf))
* downgrade range-derived mapping entries ([0ca40e2](https://github.com/io41/vibe-xpls/commit/0ca40e2c0aa15ca8333f37a6ef996fe30a1eee39))
* downgrade range-derived yaml entries ([9ff34cd](https://github.com/io41/vibe-xpls/commit/9ff34cd43801e74dbc7be3150ef3663a36ebd78e))
* expose yaml path occurrences ([99cc719](https://github.com/io41/vibe-xpls/commit/99cc7195424c50f8c3f9e39ba0718efb579b09b9))
* find template-derived path occurrences ([e5dbb82](https://github.com/io41/vibe-xpls/commit/e5dbb828da3e4cbe6baa5a4e5de1215bc0a4c31d))
* handle dash-only sequence sniffing ([b5c2414](https://github.com/io41/vibe-xpls/commit/b5c241423865dcb91f682f1dfff6e4046573b532))
* handle source position edge cases ([b5a5070](https://github.com/io41/vibe-xpls/commit/b5a5070786959a0b4a9d5e4b0a91ee2a538deb13))
* harden lsp message framing ([c1835c4](https://github.com/io41/vibe-xpls/commit/c1835c40c9648bb2e8ab8a772d439cd3a6be2601))
* harden mixed yaml template parsing ([da0c20a](https://github.com/io41/vibe-xpls/commit/da0c20a6dc1ce4865416ada742dd4a84c9261381))
* ignore block scalar shape sniffing ([bf943e2](https://github.com/io41/vibe-xpls/commit/bf943e2ea0c7585098379254318370ddc0d9ac3b))
* ignore sequence shape sniffing ([89cb13f](https://github.com/io41/vibe-xpls/commit/89cb13f8fb5f822fa95a5fa2b2e13e9d6e3f9f8c))
* pin first release version ([39944ad](https://github.com/io41/vibe-xpls/commit/39944ad0f92fad7864a82b573f2dcc08452042cb))
* prefer latest duplicate yaml values ([8f6abd6](https://github.com/io41/vibe-xpls/commit/8f6abd6c0d4474f7c8cfed2dfe83aaf71b243d03))
* prefer yaml path occurrences ([eab90bd](https://github.com/io41/vibe-xpls/commit/eab90bd76937c93685586cb405442edaf345f943))
* preserve completion edit ranges ([b4a42db](https://github.com/io41/vibe-xpls/commit/b4a42db7408f181398ab864417e6c9bf0c05f90b))
* preserve stable yaml prefix paths ([2ec2fe7](https://github.com/io41/vibe-xpls/commit/2ec2fe7f48459864f13e86f09070d67444f2dfba))
* reject standalone output template values ([2d91ce4](https://github.com/io41/vibe-xpls/commit/2d91ce4a058b1862d8fbc446d6a1fa8e02a643ba))
* reject templated block scalar paths ([ba00bdf](https://github.com/io41/vibe-xpls/commit/ba00bdfb66319340dccd9b2e12353e9f119812a5))
* reject unterminated template scalar paths ([a864a1a](https://github.com/io41/vibe-xpls/commit/a864a1aaaac0808332e139fb8ca117b743de099f))
* require shape for no-root kind activation ([32cfc24](https://github.com/io41/vibe-xpls/commit/32cfc24fa3389e873fe13844236acf2e1c071cd1))
* scope range index poisoning ([2e8efa9](https://github.com/io41/vibe-xpls/commit/2e8efa921e32e6459e5d252789d65061df7d37c5))
* sniff no-root malformed shape activation ([02f741f](https://github.com/io41/vibe-xpls/commit/02f741f2eecbf8dfcd5e23f290809e2023b439a9))
* sniff oversized no-root activation ([fa4f4e4](https://github.com/io41/vibe-xpls/commit/fa4f4e43cf21d4b43814f9081214bb39e9aa2ef7))
* suppress inactive oversized diagnostics ([52e9075](https://github.com/io41/vibe-xpls/commit/52e907513826728aa19803431cf88374305608da))
* track template-output range indexes ([39ed4db](https://github.com/io41/vibe-xpls/commit/39ed4db07ade509de507ee4013339bb44d394d9f))
* use document-local root values ([4565299](https://github.com/io41/vibe-xpls/commit/4565299a2e6b61d9dd82a09b621c9fa409b75929))
* wire app stdin and parser pins ([6dab5dd](https://github.com/io41/vibe-xpls/commit/6dab5dd06fa5ada6aee617e674f74fb8d06105ba))


### Documentation

* add crossplane lsp research program design ([4202e1a](https://github.com/io41/vibe-xpls/commit/4202e1a615e39f9edd2f8ffee61540e74ad39894))
* add product and workflow research lanes ([cdd521c](https://github.com/io41/vibe-xpls/commit/cdd521ce5736431e5aefe51ed936803e577c168c))
* add technical research lanes ([ea623da](https://github.com/io41/vibe-xpls/commit/ea623da2ead062e5d9e85d5783ef6174c45818c8))
* add zed replacement spike report ([d2ec7d4](https://github.com/io41/vibe-xpls/commit/d2ec7d46474acd88d786a893eb56a5b7e09796ad))
* add zed validation checklist ([43afb1a](https://github.com/io41/vibe-xpls/commit/43afb1ae4c6de72d434c36812305327f557e9ece))
* address first milestone plan review ([5e64ed7](https://github.com/io41/vibe-xpls/commit/5e64ed73b33e3270253bcc5d70ca2615e429b6f9))
* address first milestone review ([38c3e0a](https://github.com/io41/vibe-xpls/commit/38c3e0a1f19b151d430ed798ab5b9e81f92902aa))
* address product research review findings ([71eb94b](https://github.com/io41/vibe-xpls/commit/71eb94be8c5a5a24683874ebb1ed9907ca7173c3))
* address research review findings ([46a683d](https://github.com/io41/vibe-xpls/commit/46a683d06f3b54dff0224f13ec98c91d770abb57))
* address synthesis review findings ([732a17e](https://github.com/io41/vibe-xpls/commit/732a17e692f2455b3dc9cf669d44d90976940cd2))
* address technical research review findings ([d9347e0](https://github.com/io41/vibe-xpls/commit/d9347e0f4aec0f48ade0da272a1d7f4a3b19c0f7))
* address zed spike review findings ([afa7db4](https://github.com/io41/vibe-xpls/commit/afa7db475a1be60f9ed7798d56ffd622a73f4474))
* decide first milestone parser ([b040609](https://github.com/io41/vibe-xpls/commit/b0406095e111134ab9e6515f41f0c5538587a237))
* design first runnable lsp milestone ([93b4a75](https://github.com/io41/vibe-xpls/commit/93b4a752a5924cf5814576f3b29e30d4e9b10ff6))
* plan completion text edits ([14d8800](https://github.com/io41/vibe-xpls/commit/14d8800d49f4f9832ab55c5f8b57507c11aa4b32))
* plan crossplane lsp research program ([2e81a1f](https://github.com/io41/vibe-xpls/commit/2e81a1f5d1161dd0da77b849ae0ddec785e7a65e))
* plan first runnable lsp milestone ([a541463](https://github.com/io41/vibe-xpls/commit/a541463124cc1e358eb59edec4cf2de10f1f91e3))
* record first runnable zed validation ([2ee8bb4](https://github.com/io41/vibe-xpls/commit/2ee8bb48f738b39468062df9e84ad6f2fa3e683d))
* record zed manual validation results ([a223a88](https://github.com/io41/vibe-xpls/commit/a223a8894c8b073a813350b182fe5a10b0c36268))
* record zed trust-gated validation attempt ([d5c3e09](https://github.com/io41/vibe-xpls/commit/d5c3e0998a3f03d9a0b0ce8af6ccaca8cd8fcd3d))
* record zed-xpls-vibe hover validation ([1541c36](https://github.com/io41/vibe-xpls/commit/1541c3628c912485a9fcc14f73c6903c3474ac61))
* scaffold research program artifacts ([28f2ab8](https://github.com/io41/vibe-xpls/commit/28f2ab878d5fcbe584887756967c9d51a7c59201))
* synthesize crossplane lsp research ([bc93ce5](https://github.com/io41/vibe-xpls/commit/bc93ce5ce85eacdc78a78a2507ecda74cfbcc248))


### Tests

* add lsp harness spike ([3b03994](https://github.com/io41/vibe-xpls/commit/3b039948b88be2d818421fb0157a1b5379bdbe6b))
* add parsing schema and agent api spikes ([39a0386](https://github.com/io41/vibe-xpls/commit/39a038687f832cb85f36863032280dba82922f0f))
* add render kubernetes and release spikes ([fecb924](https://github.com/io41/vibe-xpls/commit/fecb92415cd46199c7eba683e6a5e400af257f60))
* address lsp harness review findings ([07ac3b2](https://github.com/io41/vibe-xpls/commit/07ac3b2c84cf08fc65f56c7ab6518884ac0a9b51))
* address task 6 review findings ([e2ac5cf](https://github.com/io41/vibe-xpls/commit/e2ac5cfaf4a632b89e7ababf5854bc7af3dd740a))
* address task 7 review findings ([35b28df](https://github.com/io41/vibe-xpls/commit/35b28df9b2bba43a1730f451ea0634b3179c85f3))
* fix agent api envelope wire format ([d18d756](https://github.com/io41/vibe-xpls/commit/d18d7564b99a157f1994aa7d57b94a03c5425b25))

## Changelog

All notable changes to this project will be documented in this file.

This project uses [Release Please](https://github.com/googleapis/release-please) to maintain this file from Conventional Commits.
