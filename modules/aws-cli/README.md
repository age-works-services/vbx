# AWS CLI

Select the module in user or project configuration:

```yaml
version: 1
modules:
  - aws-cli
```

The module downloads the latest AWS CLI v2 Linux x86_64 ZIP and detached
signature directly from `https://awscli.amazonaws.com/`. It verifies the archive
before extracting or executing it, using a private temporary GnuPG home and the
bundled AWS CLI Team signing key:

```text
FB5D B77F D5C1 18B8 0511 ADA8 A631 0ACC 4672 475C
```

The key comes from the [official AWS installation instructions](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html).
Key rotation or expiration requires updating the bundled key from AWS and checking
its fingerprint. Verification failures stop installation.

The combined VBX APT phase supplies download, signature-verification, extraction,
and CLI help dependencies. The module runs AWS's installer with guest sudo,
installing under `/usr/local/aws-cli` with commands under `/usr/local/bin`.
`--update` supports fresh installations and retries. Setup checks `aws --version`
for v2 and removes its temporary downloads and keyring on success or failure.

VBX runs setup during creation or an incomplete provisioning retry. Recreate the
VM to pick up a newer AWS release after successful provisioning.

## Tests

AWS-specific tests live in this directory. Run them from the repository root:

```sh
go test ./modules/aws-cli
bash modules/aws-cli/test/test-aws-cli.sh
```

`make test` runs both. The offline shell suite requires Bash, GnuPG, and standard
Linux utilities. It validates the bundled signing key, then substitutes local
fixtures for downloads, verification outcomes, extraction, guest sudo, the AWS
installer, and the installed CLI. Cases cover fresh installation, repeated
installation, retry after partial installation, stage failures, wrong CLI major
version, ordering, and temporary-file cleanup. Tests leave host packages,
installation directories, and personal keyrings unchanged.

For real guest acceptance, create a disposable VM with `aws-cli` selected, run
`vbx exec aws --version` and `vbx exec aws configure list-profiles`, then verify
`vbx exec aws help` opens the help renderer. These checks need no AWS credentials.
