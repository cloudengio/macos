# Package [cloudeng.io/macos/keychain](https://pkg.go.dev/cloudeng.io/macos/keychain?tab=doc)

```go
import cloudeng.io/macos/keychain
```

Package keychain provides support for working with the macos keychain.

## Constants
### AccessibleDefault, AccessibleWhenUnlocked, AccessibleAfterFirstUnlock, AccessibleAlways, AccessibleWhenPasscodeSetThisDeviceOnly, AccessibleWhenUnlockedThisDeviceOnly, AccessibleAfterFirstUnlockThisDeviceOnly, AccessibleAccessibleAlwaysThisDeviceOnly
```go
AccessibleDefault = keychain.AccessibleDefault
AccessibleWhenUnlocked = keychain.AccessibleWhenUnlocked
AccessibleAfterFirstUnlock = keychain.AccessibleAfterFirstUnlock
AccessibleAlways = keychain.AccessibleAlways
AccessibleWhenPasscodeSetThisDeviceOnly = keychain.AccessibleWhenPasscodeSetThisDeviceOnly
AccessibleWhenUnlockedThisDeviceOnly = keychain.AccessibleWhenUnlockedThisDeviceOnly
AccessibleAfterFirstUnlockThisDeviceOnly = keychain.AccessibleAfterFirstUnlockThisDeviceOnly
AccessibleAccessibleAlwaysThisDeviceOnly

```



## Types
### Type Accessiblity
```go
type Accessiblity int
```
Accessible is the items accessibility


### Type KeychainType
```go
type KeychainType int
```
KeychainType represents the type of keychain to use.

### Constants
### KeychainFileBased, KeychainDataProtectionLocal, KeychainICloud
```go
// KeychainFileBased represents the file-based keychain.
// This is the legacy, local only, file based keychain.
KeychainFileBased KeychainType = iota
// KeychainDataProtectionLocal represents the data protection
// keychain which is local, but integrated with the system's secure
// enclave. Applications that use must be signed and have
// appropriate entitlements.
KeychainDataProtectionLocal
// KeychainICloud represents the iCloud keychain that can be synced
// across devices.
// Applications that use must be signed and have appropriate
// entitlements.
KeychainICloud

```



### Functions

```go
func ParseKeychainType(s string) (KeychainType, error)
```
ParseKeychainType parses a string into a KeychainType.



### Methods

```go
func (t KeychainType) String() string
```




### Type Option
```go
type Option func(o *options)
```
Option represents an option for configuring a keychain.T

### Functions

```go
func WithAccessibility(v Accessiblity) Option
```


```go
func WithUpdateInPlace(v bool) Option
```




### Type SecureNoteReader
```go
type SecureNoteReader interface {
	ReadSecureNote(service string) (data []byte, err error)
}
```
SecureNoteReader defines the interface for reading secure notes from the
keychain.

### Functions

```go
func NewKeychainReadonly(typ KeychainType, account string, opts ...Option) SecureNoteReader
```
NewKeychainReadonly creates a new readonly Keychain.




### Type T
```go
type T struct {
	// contains filtered or unexported fields
}
```
T represents a keychain that can be used to read and write secure notes.

### Functions

```go
func NewKeychain(typ KeychainType, account string, opts ...Option) *T
```
NewKeychain creates a new Keychain.



### Methods

```go
func (kc T) ReadFileCtx(ctx context.Context, service string) ([]byte, error)
```


```go
func (kc T) ReadSecureNote(service string) ([]byte, error)
```
ReadSecureNote reads a secure note from the keychain.


```go
func (kc T) UpdateSecureNote(service string, data []byte) error
```
UpdateSecureNote updates an existing secure note in the keychain.


```go
func (kc T) WriteFileCtx(ctx context.Context, service string, data []byte) error
```


```go
func (kc T) WriteSecureNote(service string, data []byte) error
```
WriteSecureNote writes a secure note to the keychain. It will update an
existing note if it WithUpdateInPlace was set to true.







