const test = require('node:test');
const assert = require('node:assert/strict');
const { settleWithin } = require('./audio_guard.js');

test('settleWithin returns true when the task completes', async () => {
    assert.equal(await settleWithin(() => Promise.resolve(), 20), true);
});

test('settleWithin returns false when the task rejects', async () => {
    assert.equal(await settleWithin(() => Promise.reject(new Error('blocked')), 20), false);
});

test('settleWithin returns false when the task never settles', async () => {
    const startedAt = Date.now();
    assert.equal(await settleWithin(() => new Promise(() => {}), 20), false);
    assert.ok(Date.now() - startedAt < 200);
});
