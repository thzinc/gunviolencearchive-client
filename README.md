# gunviolencearchive-client

This is a tool to query the [Gun Violence Archive](https://www.gunviolencearchive.org). This program is unaffiliated with the Gun Violence Archive.

## Quickstart

Run the command line interface to get CSV results:

```bash
gva query incidents --from 2020-01-01 --to 2020-01-07
```

[Full documentation for `gva`](docs/gva/gva.md)

## Building

![build](https://github.com/thzinc/gunviolencearchive-client/workflows/build/badge.svg)

To build this software:

```bash
make build
```

To update the docs for `gva`:

```bash
make generate
```

## Code of Conduct

We are committed to fostering an open and welcoming environment. Please read our [code of conduct](CODE_OF_CONDUCT.md) before participating in or contributing to this project.

## Contributing

We welcome contributions and collaboration on this project. Please read our [contributor's guide](CONTRIBUTING.md) to understand how best to work with us.

## License and Authors

[![Daniel James logo](https://secure.gravatar.com/avatar/eaeac922b9f3cc9fd18cb9629b9e79f6.png?size=16) Daniel James](https://github.com/thzinc)

[![license](https://img.shields.io/github/license/thzinc/gunviolencearchive-client.svg)](https://github.com/thzinc/gunviolencearchive-client/blob/master/LICENSE)
[![GitHub contributors](https://img.shields.io/github/contributors/thzinc/gunviolencearchive-client.svg)](https://github.com/thzinc/gunviolencearchive-client/graphs/contributors)

This software is made available by Daniel James under the MIT license.
