import ejs from "ejs";
import { helpers, getContext, ObjectType, EnumType, ArrayType, XtpNormalizedType, MapType, Parameter, Property, XtpTyped } from "@dylibso/xtp-bindgen"

function toGolangTypeX(type: XtpNormalizedType): string {
  // turn into reference pointer if needed
  const pointerify = (t: string) => {
    return `${type.nullable ? '*' : ''}${t}`
  }

  switch (type.kind) {
    case 'string':
      return pointerify('string')
    case 'int32':
      return pointerify('int32')
    case 'float':
      return pointerify('float32')
    case 'double':
      return pointerify('float64')
    case 'byte':
      return pointerify('byte')
    case 'date-time':
      return pointerify('time.Time')
    case 'boolean':
      return pointerify('bool')
    case 'array':
      const arrayType = type as ArrayType
      return pointerify(`[]${toGolangTypeX(arrayType.elementType)}`)
    case 'buffer':
      return pointerify('[]byte')
    case 'object':
      const oType = (type as ObjectType)
      if (oType.properties?.length > 0) {
        return pointerify(goName(oType.name))
      } else {
        // let's use empty interface for an untyped object
        return pointerify("interface{}")
      }
    case 'enum':
      return pointerify(goName((type as EnumType).name))
    case 'map':
      const { keyType, valueType } = type as MapType
      return pointerify(`map[${toGolangTypeX(keyType)}]${toGolangTypeX(valueType)}`)
    default:
      throw new Error("Can't convert XTP type to Go type: " + type)
  }
}

function toGolangType(property: XtpTyped, required?: boolean): string {
  const t = toGolangTypeX(property.xtpType)

  // if required is unset, just return what we get back
  if (required === undefined) return t

  // if it's set and true, just return what we get back
  if (required) return t

  // otherwise it's false, so let's ensure it's a pointer
  if (t.startsWith('*')) return t
  return `*${t}`
}

// used to define a return type, for backwards compat
function toGolangReturnType(property: Property): string {
  const t = toGolangTypeX(property.xtpType)

  if (t.startsWith('[]') || t.startsWith('map[')) return t
  return `*${t}`
}

function makePublic(s: string) {
  const cap = s.charAt(0).toUpperCase();
  if (s.charAt(0) === cap) {
    return s;
  }

  const pub = cap + s.slice(1);
  return pub;
}

function goName(s: string) {
  if (!s) throw Error('Name missing to convert')
  return makePublic(helpers.snakeToCamelCase(s));
}

export function render() {
  const tmpl = Host.inputString();
  const ctx = {
    ...helpers,
    ...getContext(),
    toGolangType,
    toGolangReturnType,
    makePublic,
    goName,
  };

  const output = ejs.render(tmpl, ctx);
  Host.outputString(output);
}
