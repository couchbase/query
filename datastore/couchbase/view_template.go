package couchbase

const templStart = `
function (doc, meta) {
  if (meta.type != "json") return;`

const templFunctions = `
  var stringToUtf8Bytes = function (str) {
    var utf8 = unescape(encodeURIComponent(str));
    var bytes = [];
    for (var i = 0; i < str.length; ++i) {
        bytes.push(str.charCodeAt(i));
    }
    return bytes;
  };

  var indexFormattedValue = function (val) {
    if (val === null) {
      return [$null];
    } else if (typeof val == "boolean") {
      return [$boolean, val];
    } else if (typeof val == "number") {
      return [$number, val];
    } else if (typeof val == "string") {
      return [$string, stringToUtf8Bytes(val)];
    } else if (typeof val == "object") {
      if (val instanceof Array) {
        return [$array, val];
      } else {
        innerKeys = [];
        for (var k in val) {
          innerKeys.push(k);
        }
        innerKeys.sort()
        innerVals = [];
        for (var i in innerKeys) {
          innerVals.push(indexFormattedValue(val[innerKeys[i]]));
        }
        return [$object, [innerKeys, innerVals]];
      }
    } else {
        return undefined;
    }
  };`

const templExpr = `
  var $var = indexFormattedValue($path);`

const templKey = `
  var key = [$keylist];
  var pos = key.indexOf(undefined);
  if (pos == 0) {
    return;
  } else if (pos > 0) {
    key.splice(pos)
  }
`

const templEmit = `
  emit(key, null);`

const templEnd = `
}
// salt: $rnd
`

const templPrimary = `
function (doc, meta) {
  var stringToUtf8Bytes = function (str) {
    var utf8 = unescape(encodeURIComponent(str));
    var bytes = [];
    for (var i = 0; i < str.length; ++i) {
        bytes.push(str.charCodeAt(i));
    }
    return bytes;
  };

  emit([$string, stringToUtf8Bytes(meta.id)], null);
}
// salt: $rnd
`
