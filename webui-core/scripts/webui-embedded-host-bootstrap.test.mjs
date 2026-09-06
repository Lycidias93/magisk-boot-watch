import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import vm from "node:vm";

const source = fs.readFileSync(new URL("../module/webroot/embedded-host-bootstrap.js", import.meta.url), "utf8");

function makeElement() {
  return { textContent: "", className: "", hidden: false };
}

function makeContext({ hostname = "mui.kernelsu.org", moduleInfo, onExec } = {}) {
  const elements = new Map([
    ["#moduleName", makeElement()],
    ["#moduleVersion", makeElement()],
    ["#connectionBadge", makeElement()],
    ["#notice", makeElement()],
  ]);
  const location = {
    hostname,
    replaced: "",
    replace(value) { this.replaced = value; },
  };
  const nativeFetch = function nativeFetch() { return Promise.resolve({ ok: true }); };
  const ksu = {
    moduleInfo: () => moduleInfo ?? JSON.stringify({
      id: "pixel-10-pro-xl-thermal-fix",
      moduleDir: "/data/adb/modules/pixel-10-pro-xl-thermal-fix",
    }),
    exec(command, callback) { onExec?.(command, callback); },
  };
  const window = { fetch: nativeFetch, ksu };
  const context = {
    window,
    location,
    document: { querySelector: selector => elements.get(selector) || null },
    console,
  };
  vm.createContext(context);
  return { context, window, location, elements, nativeFetch };
}

test("embedded KsuWebUI host starts only the fixed module launcher and redirects to loopback", () => {
  let execCommand = "";
  let callback = "";
  const env = makeContext({
    onExec(command, callbackName) {
      execCommand = command;
      callback = callbackName;
    },
  });

  vm.runInContext(source, env.context);

  assert.equal(
    execCommand,
    "if [ -r '/data/adb/modules/pixel-10-pro-xl-thermal-fix/tools/webui/launch.sh' ]; then /system/bin/sh '/data/adb/modules/pixel-10-pro-xl-thermal-fix/tools/webui/launch.sh' --print-url; else /system/bin/sh '/data/adb/modules/pixel-10-pro-xl-thermal-fix/action.sh' --print-url; fi",
  );
  assert.equal(callback, "window.__rootModuleWebuiEmbeddedBootstrapDone");
  assert.notEqual(env.window.fetch, env.nativeFetch);

  const token = "a".repeat(64);
  env.window.__rootModuleWebuiEmbeddedBootstrapDone(
    0,
    `WEBUI_BOOTSTRAP_URL=http://127.0.0.1:43969/bootstrap?token=${token}\nRESULT: WEBUI_ACTION_URL_DONE outcome=success command_exit_code=0 workflow_exit_code=0\n`,
    "",
  );

  assert.equal(env.location.replaced, `http://127.0.0.1:43969/bootstrap?token=${token}`);
  assert.equal(env.window.__rootModuleWebuiEmbeddedBootstrapDone, undefined);
});

test("embedded bootstrap rejects unsafe module paths before root execution", () => {
  let execCount = 0;
  const env = makeContext({
    moduleInfo: JSON.stringify({
      id: "pixel-10-pro-xl-thermal-fix",
      moduleDir: "/data/adb/modules/pixel-10-pro-xl-thermal-fix;touch /data/local/tmp/oops",
    }),
    onExec() { execCount += 1; },
  });

  vm.runInContext(source, env.context);

  assert.equal(execCount, 0);
  assert.equal(env.elements.get("#connectionBadge").textContent, "disconnected");
  assert.match(env.elements.get("#notice").textContent, /invalid module path/i);
});

test("non-embedded loopback/browser origin does not use the KsuWebUI bridge", () => {
  let execCount = 0;
  const env = makeContext({ hostname: "127.0.0.1", onExec() { execCount += 1; } });

  vm.runInContext(source, env.context);

  assert.equal(execCount, 0);
  assert.equal(env.window.fetch, env.nativeFetch);
  assert.equal(env.location.replaced, "");
});
