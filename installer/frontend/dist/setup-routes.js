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
// downgrade differ only in wording and in which button is the safe one.
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
    const live = () => backend().SetShortcuts(read('startMenu'), read('desktop'))
    const read = renderOptions($('manage-options'), [
        {
            key: 'startMenu', label: 'Add a Start Menu entry',
            checked: state.startMenu, onChange: () => live(),
        },
        {
            key: 'desktop', label: 'Add a Desktop shortcut',
            checked: state.desktop, onChange: () => live(),
        },
        {
            key: 'boot', label: 'Start it when I sign in',
            hint: 'It waits quietly in the notification area until the game runs.',
            checked: state.launchOnBoot,
            onChange: (on) => backend().SetLaunchOnBoot(on),
        },
        launchOption(),
    ])
    showScreen('manage')
    setFooter([
        {label: 'Uninstall', kind: 'danger', onClick: () => routeUninstall(state)},
        {label: 'Close', onClick: () => backend().Quit()},
        {
            label: 'Reinstall', onClick: () => finish(
                () => backend().Install(freshChoices), read('launch'),
                `Reinstalling ${appName}`, `${appName} is reinstalled`,
                'The files were written again and the shortcuts put back as a new install would leave them.'),
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
    setFooter([
        {label: 'Cancel', onClick: () => state.installed ? route(state) : backend().Quit()},
        {
            label: 'Uninstall', kind: 'danger',
            onClick: () => withAppClosed(() => run(
                () => backend().Uninstall(read('state')),
                `Removing ${appName}`, `${appName} is removed`,
                'The application and its shortcuts are gone.')),
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
    $('brand').textContent = `${appName} Setup`
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
