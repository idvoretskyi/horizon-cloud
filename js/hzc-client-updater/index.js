const os = require('os');
const path = require('path');
const fs = require('mz/fs');
const rp = require('request-promise');
const crypto = require('crypto');
const child_process = require('mz/child_process');

const goArch = {
    x64: "amd64",
    ia32: "386",
}[os.arch()];

const goOS = {
    linux: "linux",
    darwin: "darwin",
}[os.platform()];

const versionCheckInterval = 86400;

// TODO: generalize or find a library for windows support
const stateDir = path.join(process.env.XDG_CONFIG_HOME || process.env.HOME, ".config", "horizon-cloud-client");
const binPath = path.join(stateDir, "hzc-client");
const versionPath = path.join(stateDir, "hzc-client.version");

function ensureDir(dir) {
    return fs.mkdir(dir).catch(function (error) {
        if (error.code == 'EEXIST') {
            return true;
        } else {
            throw error;
        }
    });
}

function ensureStateDir() {
    return ensureDir(path.dirname(stateDir)).then(function () {
        return ensureDir(stateDir);
    });
}

function getVersion() {
    return fs.readFile(versionPath).then(function (value) {
        return parseInt(value, 10);
    }, function (error) {
        if (error.code == 'ENOENT') {
            return 0; // 0 is less than all real versions.
        } else {
            throw error;
        }
    });
}

// Returns Promise<number> of the number of seconds old the version file is. If
// there is no version file, the promise is fulfilled with a negative value.
function getVersionAge() {
    return fs.stat(versionPath).then(function (stat) {
        return (new Date() - stat.mtime) / 1000;
    }, function (error) {
        if (error.code == 'ENOENT') {
            return -1;
        } else {
            throw error;
        }
    });
}

function getUpdateMetadata() {
    return rp({uri: "https://update.hzc.io/metadata.json"}).then(JSON.parse);
}

// ensureUpdatedBinary makes sure there is a binary at binPath of the latest
// version. It uses a .version file next to the binary to avoid checking too
// often and also to avoid downloading a version we already have, so it's safe
// to call very often.
//
// It returns a Promise<boolean> denoting whether an update actually occurred or
// not. If it can't check for an update or has trouble downloading an update,
// the Promise returned will be rejected.
function ensureUpdatedBinary() {
    return getVersionAge().then(function (age) {
        if (age >= 0 && age < versionCheckInterval) {
            return false;
        } else {
            return forceCheckUpdates();
        }
    });
}

// forceCheckUpdates is the workhorse of ensureUpdatedBinary and does the same
// job, but sends a request to the update server every time it's called. If the
// hzc-client binary indicates that it must be updated before it can continue,
// call this function and retry once.
//
// Its return value has the same format as ensureUpdatedBinary.
function forceCheckUpdates() {
    return Promise.all([
        getUpdateMetadata(),
        getVersion(),
    ]).then(function (values) {
        const metadata = values[0];
        const myVersion = values[1];

        const latestVersion = metadata["hzc-client"]["version"];

        if (latestVersion <= myVersion) {
            // Already up to date. Touch our version file to avoid checking
            // for another update until maxVersionAge has passed.
            //
            // TODO: Should we validate the hash of the binary on disk at
            // this point?
            return fs.open(versionPath, 'a+').then(function (fd) {
                return fs.futimes(fd, new Date(), new Date()).then(function(){
                    return fs.close(fd);
                });
            }).then(function () { return false });
        }

        const binMetadata = metadata["hzc-client"]["binaries"][goOS + "-" + goArch];
        if (binMetadata === undefined) {
            throw new Error("Unsupported platform for hzc-client");
        }

        console.log("Downloading new horizon cloud client...");

        return rp({
            uri: binMetadata.url,
            encoding: null, // return body as a Buffer, do not decode
        }).then(function (binary) {
            const hasher = crypto.createHash('sha256');
            hasher.update(binary);
            const hash = hasher.digest('hex');
            if (hash != binMetadata.sha256) {
                throw new Error("Bad SHA256 for binary");
            }
            return binary;
        }).then(function (binary) {
            return fs.writeFile(binPath, binary);
        }).then(function () {
            return fs.chmod(binPath, 0755);
        }).then(function () {
            // NB: We must write the version file last so that if something goes
            // wrong, we'll retry the download instead of using a stale version.
            return fs.writeFile(versionPath, "" + latestVersion);
        }).then(function () {
            return true;
        });
    });
}

function runBin(args) {
    return new Promise(function (fulfill, reject) {
        const ch = child_process.spawn(binPath, args, {
            stdio: 'inherit',
            env: Object.assign(process.env, {"HZC_FROM_HZ": "1"}),
        });
        ch.on('close', function (code) {
            if (code != 0) {
                reject(new Error(binPath + " exited with code " + code));
            } else {
                fulfill(true);
            }
        });
    });
}

// runHzcClient takes an array of arguments to the hzc client, then updates and
// runs the hzc client with those arguments.
//
// Returns a Promise which is rejected if anything goes wrong (including
// errors updating the binary and errors while running the binary.)
function runHzcClient(args) {
    return ensureStateDir().then(function () {
        return ensureUpdatedBinary();
    }).then(function () {
        return runBin(args);
    }).catch(function (execError) {
        // On error, check for updates (up to once every 5 minutes.) If an
        // update was found, retry the execution.
        return getVersionAge().then(function (age) {
            if (age >= 0 && age < 300) {
                throw execError;
            }

            return forceCheckUpdates().then(function (updated) {
                if (!updated) {
                    throw execError;
                }

                return runBin(args);
            });
        });
    });
};

module.exports = {
    runHzcClient: runHzcClient,
};
