/* ----------------------------------------------------------------- routes */

function routeInstall(state) {
    $('install-title').textContent = `Install ${appName} ${state.thisVersion}`
    $('install-path').textContent = state.installDir
    const read = renderOptions($('install-options'), shortcutOptions(state).concat([
        launchOption(),
    ]))
    showScreen('install')
    setFooter([
        {label: 'Cancel', onClick: () => backend().Quit()},
        {
            label: 'Install', kind: 'primary',
            onClick: () => install(read, `Installing ${appName}`,
                `${appName} is installed`,
                'You are on v' + state.thisVersion + '.'),
        },
    ])
}

// routeChange serves both directions of a version change, because an update and a
// downgrade differ only in wording. Either way the change itself leads: whoever ran an
// older setup file over a newer install did so to go back.
function routeChange(state) {
    const goingBack = state.relation === 'older'
    $('update-title').textContent = goingBack ? 'Go back a version?' : 'Update available'
    $('update-lead').textContent = goingBack
        ? 'This setup file carries an older version than the one installed. Your recordings and settings are untouched.'
        : 'A newer version is ready to install. Your recordings and settings are untouched.'
    $('update-from').textContent = 'v' + state.installedVersion
    $('update-to').textContent = 'v' + state.thisVersion
    const read = renderOptions($('update-options'), shortcutOptions(state).concat([
        launchOption(),
    ]))
    showScreen('update')
    setFooter([
        {label: 'Uninstall', kind: 'danger', onClick: () => routeUninstall(state)},
        {label: 'Close', onClick: () => backend().Quit()},
        {
            label: goingBack ? 'Go back' : 'Update', kind: 'primary',
            onClick: () => install(read,
                goingBack ? 'Going back a version' : `Updating ${appName}`,
                goingBack ? 'Version changed' : `${appName} is updated`,
                'You are on v' + state.thisVersion + '.'),
        },
    ])
}

// routeManage is the screen for a matching version. Its boxes act immediately,
// so closing setup from here has already applied them.
function routeManage(state) {
    $('manage-title').textContent = `${appName} ${state.installedVersion} is installed`
    // A box that saves at once and fails is put back and says why, so no box stands
    // ticked over a choice that was never saved (FR-235).
    const undo = (box, on, error) => {
        box.checked = !on
        showError(String(error))
    }
    const live = (on, box) => backend().SetShortcuts(read('startMenu'), read('desktop')).catch((e) => undo(box, on, e))
    const read = renderOptions($('manage-options'), [
        {
            key: 'startMenu', label: 'Add a Start Menu entry',
            checked: state.startMenu, onChange: (on, box) => live(on, box),
        },
        {
            key: 'desktop', label: 'Add a Desktop shortcut',
            checked: state.desktop, onChange: (on, box) => live(on, box),
        },
        {
            key: 'boot', label: 'Start it when I sign in',
            hint: 'It waits quietly in the notification area until the game runs.',
            checked: state.launchOnBoot,
            onChange: (on, box) => backend().SetLaunchOnBoot(on).catch((e) => undo(box, on, e)),
        },
        launchOption(),
    ])
    showScreen('manage')
    setFooter([
        {label: 'Uninstall', kind: 'danger', onClick: () => routeUninstall(state)},
        {label: 'Close', onClick: () => backend().Quit()},
        {
            // FR-235: the boxes as they stand, never a set of choices of its own.
            label: 'Reinstall', onClick: () => finish(
                () => backend().Install({
                    startMenu: read('startMenu'),
                    desktop: read('desktop'),
                    launchOnBoot: read('boot'),
                }), read('launch'),
                `Reinstalling ${appName}`, `${appName} is reinstalled`,
                'The files were written again with the boxes above applied as they stood.'),
        },
        {
            label: 'Repair', kind: 'primary',
            onClick: () => finish(
                () => backend().Repair(), read('launch'),
                `Repairing ${appName}`, 'Repair complete',
                'The files have been put back and nothing else was changed.'),
        },
    ])
}

function routeUninstall(state) {
    const read = renderOptions($('uninstall-options'), [
        {
            key: 'state', label: 'Also forget my settings',
            hint: 'Removes the folders you chose, the voice you cast, the theme and the volume. Your recordings are not touched. It cannot be undone.',
            checked: false,
        },
    ])
    showScreen('uninstall')
    // Cancel goes back to the screen setup opened on. Opened from the Apps list, that
    // screen is this one, so there is nothing to go back to: setup closes and leaves the
    // reader on the list they came from (FR-805).
    setFooter([
        {label: 'Cancel', onClick: () => state.mode === 'manage' ? route(state) : backend().Quit()},
        {
            label: 'Uninstall', kind: 'danger',
            onClick: () => withAppClosed(() => run(
                () => backend().Uninstall(read('state')),
                `Removing ${appName}`, `${appName} is removed`,
                'The application, its shortcuts, its log and the lines made for machine voices are gone.')),
        },
    ])
}


function route(state) {
    currentState = state
    if (state.mode === 'uninstall') {
        routeUninstall(state)
    } else if (state.mode !== 'manage') {
        routeInstall(state)
    } else if (state.relation === 'same') {
        routeManage(state)
    } else {
        routeChange(state)
    }
}

async function init() {
    applyTheme('light')
    let tries = 0
    while (!backend() && tries < 100) {
        await new Promise((resolve) => setTimeout(resolve, 50))
        tries++
    }
    if (!backend()) {
        $('error-msg').textContent = 'Could not reach the setup program.'
        showScreen('error')
        return
    }
    window.runtime.EventsOn('progress', onProgress)
    const state = await backend().DetectState()
    appName = state.appName
    document.title = `${appName} Setup`
    $('uninstall-title').textContent = `Remove ${appName}?`
    $('running-title').textContent = `${appName} is open`
    applyTheme(state.prefersDark ? 'dark' : 'light')
    route(state)
    settleKeyboard()
}

// The drawn mark is the fallback. A real icon.png beside this page replaces it;
// a missing one leaves the drawing rather than a broken image, so the header is
// right either way.
//
// The drawing is REMOVED rather than hidden. An SVG element is not an
// HTMLElement, so setting .hidden on it assigns a plain JavaScript property and
// reaches the document not at all: no attribute, no change of display. That is
// measured; it is why both marks once appeared side by side.
//
// The image may also have finished loading before this runs, in which case no
// load event is ever fired, so the already-complete case is handled directly.
const markImage = $('markimg')
const showImage = () => {
    markImage.hidden = false
    $('mark').remove()
}
if (markImage.complete && markImage.naturalWidth > 0) {
    showImage()
} else {
    markImage.onload = showImage
    markImage.onerror = () => { markImage.remove() }
}

window.addEventListener('DOMContentLoaded', init)
