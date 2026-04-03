import { deepEqual } from "./deepEqual";

describe("deepEqual", () => {
  it("returns true for identical primitives", () => {
    expect(deepEqual(1, 1)).toBe(true);
    expect(deepEqual("a", "a")).toBe(true);
    expect(deepEqual(true, true)).toBe(true);
    expect(deepEqual(null, null)).toBe(true);
    expect(deepEqual(undefined, undefined)).toBe(true);
  });

  it("returns false for different primitives", () => {
    expect(deepEqual(1, 2)).toBe(false);
    expect(deepEqual("a", "b")).toBe(false);
    expect(deepEqual(null, undefined)).toBe(false);
  });

  it("compares objects regardless of key order", () => {
    expect(deepEqual({ a: 1, b: 2 }, { b: 2, a: 1 })).toBe(true);
  });

  it("detects different object values", () => {
    expect(deepEqual({ a: 1 }, { a: 2 })).toBe(false);
  });

  it("detects different object keys", () => {
    expect(deepEqual({ a: 1 }, { b: 1 })).toBe(false);
    expect(deepEqual({ a: 1 }, { a: 1, b: 2 })).toBe(false);
  });

  it("detects different keys even when values are undefined", () => {
    expect(deepEqual({ x: undefined }, { y: undefined })).toBe(false);
  });

  it("compares nested objects", () => {
    expect(deepEqual({ a: { b: { c: 1 } } }, { a: { b: { c: 1 } } })).toBe(
      true
    );
    expect(deepEqual({ a: { b: { c: 1 } } }, { a: { b: { c: 2 } } })).toBe(
      false
    );
  });

  it("compares arrays", () => {
    expect(deepEqual([1, 2, 3], [1, 2, 3])).toBe(true);
    expect(deepEqual([1, 2], [1, 2, 3])).toBe(false);
    expect(deepEqual([1, 2, 3], [1, 3, 2])).toBe(false);
  });

  it("handles mixed types", () => {
    expect(deepEqual({ a: 1 }, [1])).toBe(false);
    expect(deepEqual(1, "1")).toBe(false);
    expect(deepEqual(null, {})).toBe(false);
  });

  it("distinguishes objects from arrays in both directions", () => {
    expect(deepEqual({}, [])).toBe(false);
    expect(deepEqual([], {})).toBe(false);
    expect(deepEqual({ 0: 1 }, [1])).toBe(false);
  });
});
