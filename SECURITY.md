# Security policy

## Supported versions

Only the latest release is supported. mangabind is a single binary with no
service behind it: update by running the install command again (see the
[README](README.md#update)).

## What counts

mangabind reads folders and `.cbz` files that people download from places they
do not control, and writes `.cbz` files. A report is welcome when a manga folder
or archive made by someone else can make mangabind:

- read or copy a file from outside the folder it was given;
- write outside the output folder;
- crash, hang, or use unbounded time, memory or disk;
- put something into a volume that the person did not choose;
- or when a released binary, an installer script or a release workflow can be
  made to run something it should not.

Bugs that only produce a wrong grouping or a wrong volume number are ordinary
issues.

## How to report

Please do not open a public issue for a vulnerability. Use GitHub's private
reporting: **Security → Report a vulnerability** on
<https://github.com/gustavommcv/mangabind/security/advisories/new>.

Say which version you ran (`mangabind --version`), the smallest folder or
archive that shows it, and what you expected. This is a one-person project: I
read reports as soon as I can, and I will tell you what I decided and when a
fixed release is out. When it is fixed the release notes say so, and I am glad
to credit you if you want.

## What is already checked

Every push, every pull request and a weekly schedule run `govulncheck` against
the code and the Go standard library it is built with; releases are built only
from a commit on `main` after the same checks pass there. See
[`docs/adr/0014-links-stay-inside-the-input.md`](docs/adr/0014-links-stay-inside-the-input.md)
for the rule about symbolic links.
