// Copyright 2015 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package parser

import (
	"bytes"
	"fmt"

	"github.com/cockroachdb/cockroach/pkg/sql/pgwire/pgerror"

	"github.com/lib/pq/oid"
)

// Type represents a SQL type.
type Type interface {
	fmt.Stringer
	// Equivalent returns whether the receiver and the other type are equivalent.
	// We say that two type patterns are "equivalent" when they are structurally
	// equivalent given that a wildcard is equivalent to any type. When neither
	// Type is ambiguous (see IsAmbiguous), equivalency is the same as type equality.
	Equivalent(other Type) bool
	// FamilyEqual returns whether the receiver and the other type have the same
	// constructor.
	FamilyEqual(other Type) bool

	// Oid returns the type's Postgres object ID.
	Oid() oid.Oid
	// SQLName returns the type's SQL standard name. This can be looked up for a
	// type `t` in postgres by running `SELECT format_type(t::regtype, NULL)`.
	SQLName() string

	// IsAmbiguous returns whether the type is ambiguous or fully defined. This
	// is important for parameterized types to determine whether they are fully
	// concrete type specification or not.
	IsAmbiguous() bool
}

const (
	fixedSize    = false
	variableSize = true
)

var (
	// TypeNull is the type of a DNull. Can be compared with ==.
	TypeNull Type = tNull{}
	// TypeBool is the type of a DBool. Can be compared with ==.
	TypeBool Type = tBool{}
	// TypeInt is the type of a DInt. Can be compared with ==.
	TypeInt Type = tInt{}
	// TypeFloat is the type of a DFloat. Can be compared with ==.
	TypeFloat Type = tFloat{}
	// TypeDecimal is the type of a DDecimal. Can be compared with ==.
	TypeDecimal Type = tDecimal{}
	// TypeString is the type of a DString. Can be compared with ==.
	TypeString Type = tString{}
	// TypeCollatedString is the type family of a DString. CANNOT be compared with
	// ==.
	TypeCollatedString Type = TCollatedString{}
	// TypeBytes is the type of a DBytes. Can be compared with ==.
	TypeBytes Type = tBytes{}
	// TypeDate is the type of a DDate. Can be compared with ==.
	TypeDate Type = tDate{}
	// TypeTimestamp is the type of a DTimestamp. Can be compared with ==.
	TypeTimestamp Type = tTimestamp{}
	// TypeTimestampTZ is the type of a DTimestampTZ. Can be compared with ==.
	TypeTimestampTZ Type = tTimestampTZ{}
	// TypeInterval is the type of a DInterval. Can be compared with ==.
	TypeInterval Type = tInterval{}
	// TypeJSON is the type of a DJSON. Can be compared with ==.
	TypeJSON Type = tJSON{}
	// TypeUUID is the type of a DUuid. Can be compared with ==.
	TypeUUID Type = tUUID{}
	// TypeINet is the type of a DIPAddr. Can be compared with ==.
	TypeINet Type = tINet{}
	// TypeTuple is the type family of a DTuple. CANNOT be compared with ==.
	TypeTuple Type = TTuple(nil)
	// TypeArray is the type family of a DArray. CANNOT be compared with ==.
	TypeArray Type = TArray{}
	// TypeTable is the type family of a DTable. CANNOT be compared with ==.
	TypeTable Type = TTable{}
	// TypePlaceholder is the type family of a placeholder. CANNOT be compared
	// with ==.
	TypePlaceholder Type = TPlaceholder{}
	// TypeAnyArray is the type of a DArray with a wildcard parameterized type.
	// Can be compared with ==.
	TypeAnyArray Type = TArray{TypeAny}
	// TypeAny can be any type. Can be compared with ==.
	TypeAny Type = tAny{}

	// TypeOid is the type of an OID. Can be compared with ==.
	TypeOid = tOid{oid.T_oid}
	// TypeRegClass is the type of an regclass OID variant. Can be compared with ==.
	TypeRegClass = tOid{oid.T_regclass}
	// TypeRegNamespace is the type of an regnamespace OID variant. Can be compared with ==.
	TypeRegNamespace = tOid{oid.T_regnamespace}
	// TypeRegProc is the type of an regproc OID variant. Can be compared with ==.
	TypeRegProc = tOid{oid.T_regproc}
	// TypeRegProcedure is the type of an regprocedure OID variant. Can be compared with ==.
	TypeRegProcedure = tOid{oid.T_regprocedure}
	// TypeRegType is the type of an regtype OID variant. Can be compared with ==.
	TypeRegType = tOid{oid.T_regtype}

	// TypeName is a type-alias for TypeString with a different OID. Can be
	// compared with ==.
	TypeName = wrapTypeWithOid(TypeString, oid.T_name)
	// TypeIntVector is a type-alias for a TypeIntArray with a different OID. Can
	// be compared with ==.
	TypeIntVector = wrapTypeWithOid(TArray{TypeInt}, oid.T_int2vector)
	// TypeNameArray is the type family of a DArray containing the Name alias type.
	// Can be compared with ==.
	TypeNameArray Type = TArray{TypeName}

	// TypesAnyNonArray contains all non-array types.
	TypesAnyNonArray = []Type{
		TypeBool,
		TypeInt,
		TypeFloat,
		TypeDecimal,
		TypeString,
		TypeBytes,
		TypeDate,
		TypeTimestamp,
		TypeTimestampTZ,
		TypeInterval,
		TypeUUID,
		TypeINet,
		TypeJSON,
		TypeOid,
	}
)

var (
	// Unexported wrapper types. These exist for Postgres type compatibility.
	typeInt2      = wrapTypeWithOid(TypeInt, oid.T_int2)
	typeInt4      = wrapTypeWithOid(TypeInt, oid.T_int4)
	typeFloat4    = wrapTypeWithOid(TypeFloat, oid.T_float4)
	typeVarChar   = wrapTypeWithOid(TypeString, oid.T_varchar)
	typeInt2Array = TArray{typeInt2}
	typeInt4Array = TArray{typeInt4}
)

// OidToType maps Postgres object IDs to CockroachDB types.
var OidToType = map[oid.Oid]Type{
	oid.T_anyelement:   TypeAny,
	oid.T_bool:         TypeBool,
	oid.T__bool:        TArray{TypeBool},
	oid.T_bytea:        TypeBytes,
	oid.T__bytea:       TArray{TypeBytes},
	oid.T_date:         TypeDate,
	oid.T__date:        TArray{TypeDate},
	oid.T_float4:       typeFloat4,
	oid.T__float4:      TArray{typeFloat4},
	oid.T_float8:       TypeFloat,
	oid.T__float8:      TArray{TypeFloat},
	oid.T_int2:         typeInt2,
	oid.T_int4:         typeInt4,
	oid.T_int8:         TypeInt,
	oid.T_int2vector:   TypeIntVector,
	oid.T_interval:     TypeInterval,
	oid.T__interval:    TArray{TypeInterval},
	oid.T_jsonb:        TypeJSON,
	oid.T_name:         TypeName,
	oid.T__name:        TArray{TypeName},
	oid.T_numeric:      TypeDecimal,
	oid.T__numeric:     TArray{TypeDecimal},
	oid.T_oid:          TypeOid,
	oid.T__oid:         TArray{TypeOid},
	oid.T_regclass:     TypeRegClass,
	oid.T_regnamespace: TypeRegNamespace,
	oid.T_regproc:      TypeRegProc,
	oid.T_regprocedure: TypeRegProcedure,
	oid.T_regtype:      TypeRegType,
	oid.T__text:        TArray{TypeString},
	oid.T__int2:        typeInt2Array,
	oid.T__int4:        typeInt4Array,
	oid.T__int8:        TArray{TypeInt},
	oid.T_record:       TypeTuple,
	oid.T_text:         TypeString,
	oid.T_timestamp:    TypeTimestamp,
	oid.T__timestamp:   TArray{TypeTimestamp},
	oid.T_timestamptz:  TypeTimestampTZ,
	oid.T__timestamptz: TArray{TypeTimestampTZ},
	oid.T_uuid:         TypeUUID,
	oid.T__uuid:        TArray{TypeUUID},
	oid.T_inet:         TypeINet,
	oid.T__inet:        TArray{TypeINet},
	oid.T_varchar:      typeVarChar,
	oid.T__varchar:     TArray{typeVarChar},
}

// AliasedOidToName maps Postgres object IDs to type names for those OIDs that map to
// Cockroach types that have more than one associated OID, like Int. The name
// for these OIDs will override the type name of the corresponding type when
// looking up the display name for an OID.
var aliasedOidToName = map[oid.Oid]string{
	oid.T_float4:     "float4",
	oid.T_float8:     "float8",
	oid.T_int2:       "int2",
	oid.T_int4:       "int4",
	oid.T_int8:       "int8",
	oid.T_int2vector: "int2vector",
	oid.T_text:       "text",
	oid.T_bytea:      "bytea",
	oid.T_varchar:    "varchar",
	oid.T_numeric:    "numeric",
	oid.T_record:     "record",
	oid.T__int2:      "_int2",
	oid.T__int4:      "_int4",
	oid.T__int8:      "_int8",
	oid.T__text:      "_text",
	// TODO(justin): find a better solution to this than mapping every array type.
	oid.T__float4:      "_float4",
	oid.T__float8:      "_float8",
	oid.T__bool:        "_bool",
	oid.T__bytea:       "_bytea",
	oid.T__date:        "_date",
	oid.T__interval:    "_interval",
	oid.T__name:        "_name",
	oid.T__numeric:     "_numeric",
	oid.T__oid:         "_oid",
	oid.T__timestamp:   "_timestamp",
	oid.T__timestamptz: "_timestamptz",
	oid.T__uuid:        "_uuid",
	oid.T__inet:        "_inet",
	oid.T__varchar:     "_varchar",
}

// PGDisplayName returns the Postgres display name for a given type.
func PGDisplayName(typ Type) string {
	if typname, ok := aliasedOidToName[typ.Oid()]; ok {
		return typname
	}
	return typ.String()
}

// Do not instantiate the tXxx types elsewhere. The variables above are intended
// to be singletons.
type tNull struct{}

func (tNull) String() string              { return "NULL" }
func (tNull) Equivalent(other Type) bool  { return other == TypeNull || other == TypeAny }
func (tNull) FamilyEqual(other Type) bool { return other == TypeNull }
func (tNull) Oid() oid.Oid                { return oid.T_unknown }
func (tNull) SQLName() string             { return "unknown" }
func (tNull) IsAmbiguous() bool           { return true }

type tBool struct{}

func (tBool) String() string              { return "bool" }
func (tBool) Equivalent(other Type) bool  { return UnwrapType(other) == TypeBool || other == TypeAny }
func (tBool) FamilyEqual(other Type) bool { return UnwrapType(other) == TypeBool }
func (tBool) Oid() oid.Oid                { return oid.T_bool }
func (tBool) SQLName() string             { return "boolean" }
func (tBool) IsAmbiguous() bool           { return false }

type tInt struct{}

func (tInt) String() string              { return "int" }
func (tInt) Equivalent(other Type) bool  { return UnwrapType(other) == TypeInt || other == TypeAny }
func (tInt) FamilyEqual(other Type) bool { return UnwrapType(other) == TypeInt }
func (tInt) Oid() oid.Oid                { return oid.T_int8 }
func (tInt) SQLName() string             { return "bigint" }
func (tInt) IsAmbiguous() bool           { return false }

type tFloat struct{}

func (tFloat) String() string              { return "float" }
func (tFloat) Equivalent(other Type) bool  { return UnwrapType(other) == TypeFloat || other == TypeAny }
func (tFloat) FamilyEqual(other Type) bool { return UnwrapType(other) == TypeFloat }
func (tFloat) Oid() oid.Oid                { return oid.T_float8 }
func (tFloat) SQLName() string             { return "double precision" }
func (tFloat) IsAmbiguous() bool           { return false }

type tDecimal struct{}

func (tDecimal) String() string { return "decimal" }
func (tDecimal) Equivalent(other Type) bool {
	return UnwrapType(other) == TypeDecimal || other == TypeAny
}
func (tDecimal) FamilyEqual(other Type) bool { return UnwrapType(other) == TypeDecimal }
func (tDecimal) Oid() oid.Oid                { return oid.T_numeric }
func (tDecimal) SQLName() string             { return "numeric" }
func (tDecimal) IsAmbiguous() bool           { return false }

type tString struct{}

func (tString) String() string              { return "string" }
func (tString) Equivalent(other Type) bool  { return UnwrapType(other) == TypeString || other == TypeAny }
func (tString) FamilyEqual(other Type) bool { return UnwrapType(other) == TypeString }
func (tString) Oid() oid.Oid                { return oid.T_text }
func (tString) SQLName() string             { return "text" }
func (tString) IsAmbiguous() bool           { return false }

// TCollatedString is the type of strings with a locale.
type TCollatedString struct {
	Locale string
}

// String implements the fmt.Stringer interface.
func (t TCollatedString) String() string {
	return fmt.Sprintf("collatedstring{%s}", t.Locale)
}

// Equivalent implements the Type interface.
func (t TCollatedString) Equivalent(other Type) bool {
	if other == TypeAny {
		return true
	}
	u, ok := UnwrapType(other).(TCollatedString)
	if ok {
		return t.Locale == "" || u.Locale == "" || t.Locale == u.Locale
	}
	return false
}

// FamilyEqual implements the Type interface.
func (TCollatedString) FamilyEqual(other Type) bool {
	_, ok := UnwrapType(other).(TCollatedString)
	return ok
}

// Oid implements the Type interface.
func (TCollatedString) Oid() oid.Oid { return oid.T_unknown }

// SQLName implements the Type interface.
func (TCollatedString) SQLName() string { return "text" }

// IsAmbiguous implements the Type interface.
func (t TCollatedString) IsAmbiguous() bool {
	return t.Locale == ""
}

type tBytes struct{}

func (tBytes) String() string              { return "bytes" }
func (tBytes) Equivalent(other Type) bool  { return UnwrapType(other) == TypeBytes || other == TypeAny }
func (tBytes) FamilyEqual(other Type) bool { return UnwrapType(other) == TypeBytes }
func (tBytes) Oid() oid.Oid                { return oid.T_bytea }
func (tBytes) SQLName() string             { return "bytea" }
func (tBytes) IsAmbiguous() bool           { return false }

type tDate struct{}

func (tDate) String() string              { return "date" }
func (tDate) Equivalent(other Type) bool  { return UnwrapType(other) == TypeDate || other == TypeAny }
func (tDate) FamilyEqual(other Type) bool { return UnwrapType(other) == TypeDate }
func (tDate) Oid() oid.Oid                { return oid.T_date }
func (tDate) SQLName() string             { return "date" }
func (tDate) IsAmbiguous() bool           { return false }

type tTimestamp struct{}

func (tTimestamp) String() string { return "timestamp" }
func (tTimestamp) Equivalent(other Type) bool {
	return UnwrapType(other) == TypeTimestamp || other == TypeAny
}
func (tTimestamp) FamilyEqual(other Type) bool { return UnwrapType(other) == TypeTimestamp }
func (tTimestamp) Oid() oid.Oid                { return oid.T_timestamp }
func (tTimestamp) SQLName() string             { return "timestamp without time zone" }
func (tTimestamp) IsAmbiguous() bool           { return false }

type tTimestampTZ struct{}

func (tTimestampTZ) String() string { return "timestamptz" }
func (tTimestampTZ) Equivalent(other Type) bool {
	return UnwrapType(other) == TypeTimestampTZ || other == TypeAny
}
func (tTimestampTZ) FamilyEqual(other Type) bool { return UnwrapType(other) == TypeTimestampTZ }
func (tTimestampTZ) Oid() oid.Oid                { return oid.T_timestamptz }
func (tTimestampTZ) SQLName() string             { return "timestamp with time zone" }
func (tTimestampTZ) IsAmbiguous() bool           { return false }

type tInterval struct{}

func (tInterval) String() string { return "interval" }
func (tInterval) Equivalent(other Type) bool {
	return UnwrapType(other) == TypeInterval || other == TypeAny
}
func (tInterval) FamilyEqual(other Type) bool { return UnwrapType(other) == TypeInterval }
func (tInterval) Oid() oid.Oid                { return oid.T_interval }
func (tInterval) SQLName() string             { return "interval" }
func (tInterval) IsAmbiguous() bool           { return false }

type tJSON struct{}

func (tJSON) String() string { return "jsonb" }
func (tJSON) Equivalent(other Type) bool {
	return UnwrapType(other) == TypeJSON || other == TypeAny
}
func (tJSON) FamilyEqual(other Type) bool { return UnwrapType(other) == TypeJSON }
func (tJSON) Oid() oid.Oid                { return oid.T_jsonb }
func (tJSON) SQLName() string             { return "json" }
func (tJSON) IsAmbiguous() bool           { return false }

type tUUID struct{}

func (tUUID) String() string              { return "uuid" }
func (tUUID) Equivalent(other Type) bool  { return UnwrapType(other) == TypeUUID || other == TypeAny }
func (tUUID) FamilyEqual(other Type) bool { return UnwrapType(other) == TypeUUID }
func (tUUID) Oid() oid.Oid                { return oid.T_uuid }
func (tUUID) SQLName() string             { return "uuid" }
func (tUUID) IsAmbiguous() bool           { return false }

type tINet struct{}

func (tINet) String() string              { return "inet" }
func (tINet) Equivalent(other Type) bool  { return UnwrapType(other) == TypeINet || other == TypeAny }
func (tINet) FamilyEqual(other Type) bool { return UnwrapType(other) == TypeINet }
func (tINet) Oid() oid.Oid                { return oid.T_inet }
func (tINet) SQLName() string             { return "inet" }
func (tINet) IsAmbiguous() bool           { return false }

// TTuple is the type of a DTuple.
type TTuple []Type

// String implements the fmt.Stringer interface.
func (t TTuple) String() string {
	var buf bytes.Buffer
	buf.WriteString("tuple")
	if t != nil {
		buf.WriteByte('{')
		for i, typ := range t {
			if i != 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(typ.String())
		}
		buf.WriteByte('}')
	}
	return buf.String()
}

// Equivalent implements the Type interface.
func (t TTuple) Equivalent(other Type) bool {
	if other == TypeAny {
		return true
	}
	u, ok := UnwrapType(other).(TTuple)
	if !ok || len(t) != len(u) {
		return false
	}
	for i, typ := range t {
		if !typ.Equivalent(u[i]) {
			return false
		}
	}
	return true
}

// FamilyEqual implements the Type interface.
func (TTuple) FamilyEqual(other Type) bool {
	_, ok := UnwrapType(other).(TTuple)
	return ok
}

// Oid implements the Type interface.
func (TTuple) Oid() oid.Oid { return oid.T_record }

// SQLName implements the Type interface.
func (TTuple) SQLName() string { return "record" }

// IsAmbiguous implements the Type interface.
func (t TTuple) IsAmbiguous() bool {
	for _, typ := range t {
		if typ == nil || typ.IsAmbiguous() {
			return true
		}
	}
	return false
}

// TPlaceholder is the type of a placeholder.
type TPlaceholder struct {
	Name string
}

// String implements the fmt.Stringer interface.
func (t TPlaceholder) String() string { return fmt.Sprintf("placeholder{%s}", t.Name) }

// Equivalent implements the Type interface.
func (t TPlaceholder) Equivalent(other Type) bool {
	if other == TypeAny {
		return true
	}
	u, ok := UnwrapType(other).(TPlaceholder)
	return ok && t.Name == u.Name
}

// FamilyEqual implements the Type interface.
func (TPlaceholder) FamilyEqual(other Type) bool {
	_, ok := UnwrapType(other).(TPlaceholder)
	return ok
}

// Oid implements the Type interface.
func (TPlaceholder) Oid() oid.Oid { panic("TPlaceholder.Oid() is undefined") }

// SQLName implements the Type interface.
func (TPlaceholder) SQLName() string { panic("TPlaceholder.SQLName() is undefined") }

// IsAmbiguous implements the Type interface.
func (TPlaceholder) IsAmbiguous() bool { panic("TPlaceholder.IsAmbiguous() is undefined") }

// TArray is the type of a DArray.
type TArray struct{ Typ Type }

func (a TArray) String() string { return a.Typ.String() + "[]" }

// Equivalent implements the Type interface.
func (a TArray) Equivalent(other Type) bool {
	if other == TypeAny {
		return true
	}
	if u, ok := UnwrapType(other).(TArray); ok {
		return a.Typ.Equivalent(u.Typ)
	}
	return false
}

// FamilyEqual implements the Type interface.
func (TArray) FamilyEqual(other Type) bool {
	_, ok := UnwrapType(other).(TArray)
	return ok
}

// oidToArrayOid maps scalar type Oids to their corresponding array type Oid.
var oidToArrayOid = map[oid.Oid]oid.Oid{
	oid.T_bool:        oid.T__bool,
	oid.T_bytea:       oid.T__bytea,
	oid.T_name:        oid.T__name,
	oid.T_int8:        oid.T__int8,
	oid.T_int2:        oid.T__int2,
	oid.T_int4:        oid.T__int4,
	oid.T_text:        oid.T__text,
	oid.T_oid:         oid.T__oid,
	oid.T_float4:      oid.T__float4,
	oid.T_float8:      oid.T__float8,
	oid.T_inet:        oid.T__inet,
	oid.T_varchar:     oid.T__varchar,
	oid.T_date:        oid.T__date,
	oid.T_timestamp:   oid.T__timestamp,
	oid.T_timestamptz: oid.T__timestamptz,
	oid.T_interval:    oid.T__interval,
	oid.T_numeric:     oid.T__numeric,
	oid.T_uuid:        oid.T__uuid,
}

const noArrayType = 0

// ArrayOids is a set of all oids which correspond to an array type.
var ArrayOids = map[oid.Oid]struct{}{}

func init() {
	for _, v := range oidToArrayOid {
		ArrayOids[v] = struct{}{}
	}
}

// Oid implements the Type interface.
func (a TArray) Oid() oid.Oid {
	if o, ok := oidToArrayOid[a.Typ.Oid()]; ok {
		return o
	}
	return noArrayType
}

// SQLName implements the Type interface.
func (a TArray) SQLName() string {
	return a.Typ.SQLName() + "[]"
}

// IsAmbiguous implements the Type interface.
func (a TArray) IsAmbiguous() bool {
	return a.Typ == nil || a.Typ.IsAmbiguous()
}

// TTable is the type of a DTable.
// See the comments at the start of generator_builtins.go for details.
type TTable struct {
	Cols   TTuple
	Labels []string
}

func (a TTable) String() string { return "setof " + a.Cols.String() }

// Equivalent implements the Type interface.
func (a TTable) Equivalent(other Type) bool {
	if u, ok := UnwrapType(other).(TTable); ok {
		return a.Cols.Equivalent(u.Cols)
	}
	return false
}

// FamilyEqual implements the Type interface.
func (TTable) FamilyEqual(other Type) bool {
	_, ok := UnwrapType(other).(TTable)
	return ok
}

// Oid implements the Type interface.
func (TTable) Oid() oid.Oid { return oid.T_anyelement }

// SQLName implements the Type interface.
func (TTable) SQLName() string { return "anyelement" }

// IsAmbiguous implements the Type interface.
func (a TTable) IsAmbiguous() bool {
	return a.Cols == nil || a.Cols.IsAmbiguous()
}

type tAny struct{}

func (tAny) String() string              { return "anyelement" }
func (tAny) Equivalent(other Type) bool  { return true }
func (tAny) FamilyEqual(other Type) bool { return other == TypeAny }
func (tAny) Oid() oid.Oid                { return oid.T_anyelement }
func (tAny) SQLName() string             { return "anyelement" }
func (tAny) IsAmbiguous() bool           { return true }

type tOid struct {
	oidType oid.Oid
}

func (t tOid) String() string             { return t.SQLName() }
func (t tOid) Equivalent(other Type) bool { return t.FamilyEqual(other) || other == TypeAny }
func (tOid) FamilyEqual(other Type) bool  { _, ok := UnwrapType(other).(tOid); return ok }
func (t tOid) Oid() oid.Oid               { return t.oidType }
func (t tOid) SQLName() string {
	switch t.oidType {
	case oid.T_oid:
		return "oid"
	case oid.T_regclass:
		return "regclass"
	case oid.T_regnamespace:
		return "regnamespace"
	case oid.T_regproc:
		return "regproc"
	case oid.T_regprocedure:
		return "regprocedure"
	case oid.T_regtype:
		return "regtype"
	default:
		panic(fmt.Sprintf("unexpected oidType: %v", t.oidType))
	}
}
func (tOid) IsAmbiguous() bool { return false }

// tOidWrapper is a Type implementation which is a wrapper around a Type, allowing
// custom Oid values to be attached to the Type. The Type is used by DOidWrapper
// to permit type aliasing with custom Oids without needing to create new typing
// rules or define new Datum types.
type tOidWrapper struct {
	Type
	oid oid.Oid
}

var customOidNames = map[oid.Oid]string{
	oid.T_name: "name",
}

func (t tOidWrapper) String() string {
	// Allow custom type names for specific Oids, but default to wrapped String.
	if s, ok := customOidNames[t.oid]; ok {
		return s
	}
	return t.Type.String()
}

func (t tOidWrapper) Oid() oid.Oid { return t.oid }

// wrapTypeWithOid wraps a Type with a custom Oid.
func wrapTypeWithOid(t Type, oid oid.Oid) Type {
	switch v := t.(type) {
	case tNull, tAny, tOidWrapper:
		panic(pgerror.NewErrorf(pgerror.CodeInternalError, "cannot wrap %T with an Oid", v))
	}
	return tOidWrapper{
		Type: t,
		oid:  oid,
	}
}

// UnwrapType returns the base Type type for a provided type, stripping
// a *TOidWrapper if present. This is useful for cases like type switches,
// where type aliases should be ignored.
func UnwrapType(t Type) Type {
	if w, ok := t.(tOidWrapper); ok {
		return w.Type
	}
	return t
}
