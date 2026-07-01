(function (root, factory) {
    const api = factory();
    if (typeof module === 'object' && module.exports) {
        module.exports = api;
    }
    root.AudioGuard = api;
})(typeof globalThis !== 'undefined' ? globalThis : this, function () {
    async function settleWithin(task, timeoutMs) {
        return Promise.race([
            Promise.resolve().then(task).then(() => true, () => false),
            new Promise(resolve => setTimeout(() => resolve(false), timeoutMs))
        ]);
    }

    return { settleWithin };
});
