# Sürüm Oluşturma Rehberi / Release Guide

Bu doküman, Tesla Takip Uygulaması için yeni bir sürüm oluşturma sürecini açıklar.

This document explains the process of creating a new release for the Tesla Tracking App.

## Sürüm Oluşturma Adımları / Release Creation Steps

### 1. Versiyon Güncelleme / Update Version

`pkg/version/version.go` dosyasındaki sürüm numaralarını güncelleyin. Semantic versioning kullanıyoruz: `Major.Minor.Patch`.

Update the version numbers in the `pkg/version/version.go` file. We use semantic versioning: `Major.Minor.Patch`.

Örnek / Example:
```go
const (
	// Major version
	Major = 1
	// Minor version
	Minor = 1  // Önceki: 0, Yeni: 1 - Previous: 0, New: 1
	// Patch version
	Patch = 0
)
```

### 2. Değişiklikleri Taahhüt Edin / Commit Changes

```bash
git add pkg/version/version.go
git commit -m "Bump version to v1.1.0"
```

### 3. Tag Oluşturun / Create Tag

```bash
git tag -a v1.1.0 -m "Version 1.1.0"
```

### 4. Uzak Depoya İtme / Push to Remote Repository

```bash
git push origin main
git push origin v1.1.0
```

### 5. GitHub Actions Takibi / Monitor GitHub Actions

GitHub tag'i algılayacak ve otomatik olarak release workflow'unu tetikleyecektir. Bu workflow:

1. Windows, macOS ve Linux için yürütülebilir dosyalar oluşturur
2. GitHub sürümü oluşturur
3. Derlenen dosyaları bu sürüme ekler

GitHub will detect the tag and automatically trigger the release workflow. This workflow:

1. Builds executables for Windows, macOS, and Linux
2. Creates a GitHub release
3. Attaches the compiled files to this release

## Sürüm Notları / Release Notes

Sürüm notlarını eklemek için https://github.com/tgezginis/tesla-tracking-app/releases adresine gidin ve ilgili sürümü düzenleyin.

To add release notes, go to https://github.com/tgezginis/tesla-tracking-app/releases and edit the relevant release.

## Sorun Giderme / Troubleshooting

Eğer derleme işlemi başarısız olursa, GitHub Actions sekmesindeki build loglarını kontrol edin. En yaygın hatalar:

- Eksik bağımlılıklar
- Eksik asset dosyaları (ör. icon.jpg)
- Yapılandırma hataları

If the build process fails, check the build logs in the GitHub Actions tab. The most common errors are:

- Missing dependencies
- Missing asset files (e.g., icon.jpg)
- Configuration errors 