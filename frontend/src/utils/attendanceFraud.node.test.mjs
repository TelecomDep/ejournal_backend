import test from 'node:test';
import assert from 'node:assert/strict';
import { getBrowserLocation } from './attendanceFraud.js';

const setBrowser = (getCurrentPosition, secure = true) => {
  Object.defineProperty(globalThis, 'navigator', { configurable: true, value: { geolocation: { getCurrentPosition } } });
  Object.defineProperty(globalThis, 'window', { configurable: true, value: { isSecureContext: secure } });
};

test('first GPS fix has time to complete and does not reuse old coordinates', async () => {
  let complete;
  setBrowser((success, failure, options) => {
    assert.equal(options.timeout, 15000);
    assert.equal(options.maximumAge, 0);
    complete = success;
  });
  const pending = getBrowserLocation();
  complete({ coords: { latitude: 55.02, longitude: 82.93 } });
  assert.deepEqual(await pending, { lat: 55.02, lon: 82.93 });
});

test('a denied request rejects but a later user retry can succeed', async () => {
  setBrowser((success, failure) => failure({ code: 1 }));
  await assert.rejects(getBrowserLocation(), /запрещён/);
  setBrowser((success) => success({ coords: { latitude: 55, longitude: 83 } }));
  assert.deepEqual(await getBrowserLocation(), { lat: 55, lon: 83 });
});

test('timeout is not reported as permission denial', async () => {
  setBrowser((success, failure) => failure({ code: 3 }));
  await assert.rejects(getBrowserLocation(), /слишком много времени/);
});

test('insecure origins never request location', async () => {
  setBrowser(() => assert.fail('must not request location on HTTP'), false);
  await assert.rejects(getBrowserLocation(), /HTTPS/);
});

test('null and out-of-range coordinates cannot be sent as an attendance mark', async () => {
  for (const coords of [{ latitude: null, longitude: null }, { latitude: 91, longitude: 82 }, { latitude: 55, longitude: 181 }]) {
    setBrowser((success) => success({ coords }));
    await assert.rejects(getBrowserLocation(), /некорректные координаты/);
  }
});
