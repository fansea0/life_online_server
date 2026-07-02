// 纯函数测试：buildStateBar 输入面板+delta，返回渲染结构
function buildStateBar(prevPanel, payload) {
    const dims = ["名望", "人心", "实力", "机缘"];
    const icons = {"名望":"star", "人心":"heart", "实力":"sword", "机缘":"clover"};
    const items = dims.map(d => ({
        dim: d,
        icon: icons[d],
        label: payload.panel[d] || "",
        dir: payload.delta ? payload.delta[d] : "0",
    }));
    return { items, options: payload.options || [] };
}

const assert = require('assert');
try {
    const r = buildStateBar({}, {
        panel: {"名望":"颇有声名","人心":"初得民望","实力":"可堪一用","机缘":"尚需时运"},
        delta: {"名望":"+","人心":"0","实力":"-","机缘":"0"},
        options: ["a","b","c"],
    });
    assert.strictEqual(r.items.length, 4);
    assert.strictEqual(r.items[0].label, "颇有声名");
    assert.strictEqual(r.items[0].dir, "+");
    assert.strictEqual(r.items[2].dir, "-");
    assert.strictEqual(r.options.length, 3);
    console.log("sanguo_state tests PASS");
} catch (e) {
    console.error("sanguo_state tests FAIL:", e.message);
    process.exit(1);
}

// kind -> 容器 class 映射
function kindToClass(kind) {
    const map = {"normal":"turn-normal","crisis":"turn-crisis","boon":"turn-boon","timeskip":"turn-timeskip"};
    return map[kind] || "turn-normal";
}

try {
    assert.strictEqual(kindToClass("crisis"), "turn-crisis");
    assert.strictEqual(kindToClass("boon"), "turn-boon");
    assert.strictEqual(kindToClass("timeskip"), "turn-timeskip");
    assert.strictEqual(kindToClass("normal"), "turn-normal");
    assert.strictEqual(kindToClass("boss"), "turn-normal");
    assert.strictEqual(kindToClass(undefined), "turn-normal");
    console.log("kindToClass tests PASS");
} catch (e) {
    console.error("kindToClass tests FAIL:", e.message);
    process.exit(1);
}
