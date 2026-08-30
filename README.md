# ms2pdf

[![CI](https://github.com/sukujgrg/ms2pdf/workflows/CI/badge.svg)](https://github.com/sukujgrg/ms2pdf/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/sukujgrg/ms2pdf.svg)](https://pkg.go.dev/github.com/sukujgrg/ms2pdf)

Local CLI that converts one Office (or other Graph-supported) file to PDF using Microsoft Graph `GET {driveItem}/content?format=pdf`.

- Docs: [pkg.go.dev/github.com/sukujgrg/ms2pdf](https://pkg.go.dev/github.com/sukujgrg/ms2pdf)
- CI: [github.com/sukujgrg/ms2pdf/actions](https://github.com/sukujgrg/ms2pdf/actions/workflows/ci.yml)

## Commands

```
ms2pdf login
ms2pdf logout
ms2pdf whoami
ms2pdf convert <file> [-o out.pdf] [--type ext]
```

`login` opens a browser. Any personal Microsoft account (Outlook/Hotmail) or work/school account can sign in. `convert` never starts a login. If there is no cached token, it exits and tells you to run `login`.

Type comes from the filename (`report.docx`). Use `--type` only when the suffix is missing or wrong. Output is always PDF. If `-o` is omitted, the PDF is written next to the input as `<basename>.pdf`.

Supported Graph PDF sources: doc, docx, dot, dotx, dotm, xls, xlsx, xlsm, ppt, pptx, pps, ppsx, rtf, odt, ods, odp, html, htm, md, markdown, eml, msg, epub, tif, tiff, dwg.

Not converted: csv, xlsb, pptm, pot/potx/potm, Apple pages/numbers/key.

## One-time Entra app (free)

Your Azure directory **owns** the app. It does **not** limit which emails can sign in, as long as the app is registered for personal + work accounts.

1. [Microsoft Entra admin center](https://entra.microsoft.com/) → **App registrations** → **New registration**.
2. **Supported account types:** *Accounts in any organizational directory and personal Microsoft accounts* (not “this directory only”).
3. **Authentication** → **Add a platform** → **Mobile and desktop applications** → tick **`http://localhost`**.
4. **Authentication** → **Allow public client flows** = Yes. Implicit ID tokens off. No client secret.
5. **API permissions** (delegated Microsoft Graph): `Files.ReadWrite`, `User.Read`.
6. Copy the **Application (client) ID**.

A personal Microsoft account + OneDrive does **not** need a paid Office licence. A work/school account usually needs a Microsoft 365 licence that provisions OneDrive for Business.

`login --device-code --tenant <DIRECTORY_ID>` is optional and **only** signs in work/school users of that tenant. Device code cannot sign in personal Microsoft accounts.

## Usage

```
make build
./bin/ms2pdf login --app-id <APPLICATION_CLIENT_ID>
./bin/ms2pdf convert report.docx
./bin/ms2pdf convert memo --type docx -o memo.pdf
```

`--app-id` can also be `MS2PDF_CLIENT_ID`. The first successful login stores the id in the user config dir so later commands can omit it.

Tokens are stored in the macOS Keychain (service `ms2pdf`, account `msal-cache`) via [go-secretstore](https://github.com/sukujgrg/go-secretstore). App id and tenant stay in `os.UserConfigDir()` as `ms2pdf/config.json`. An existing `msal.json` is copied into Keychain on first use, then removed. Files are uploaded to `me/drive` as a temporary item, converted, then deleted.

## Makefile

| Target | Purpose |
| --- | --- |
| `make build` | CGO-enabled `bin/ms2pdf` |
| `make test` | unit tests |
| `make vet` | `go vet` |
| `make clean` | remove `bin/` |

macOS and Linux builds need CGO because token storage uses go-secretstore.

## Licence notes

The file leaves the machine. Graph size and rate limits apply. The temp upload counts against OneDrive quota.
