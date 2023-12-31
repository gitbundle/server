import $ from 'jquery'

const $service = $('#service_type')
const $user = $('#auth_username')
const $pass = $('#auth_password')
const $token = $('#auth_token')
const $mirror = $('#mirror')
const $items = $('#migrate_items').find('input[type=checkbox]')

export function initRepoMigration() {
  checkAuth()

  $user.on('keyup', () => {
    checkItems(false)
  })
  $pass.on('keyup', () => {
    checkItems(false)
  })
  $token.on('keyup', () => {
    checkItems(true)
  })
  $mirror.on('change', () => {
    checkItems(true)
  })

  const $cloneAddr = $('#clone_addr')
  $cloneAddr.on('change', () => {
    const $repoName = $('#repo_name')
    if ($cloneAddr.val().length > 0 && $repoName.val().length === 0) {
      // Only modify if repo_name input is blank
      $repoName.val($cloneAddr.val().match(/^(.*\/)?((.+?)(\.git)?)$/)[3])
    }
  })
}

function checkAuth() {
  const serviceType = $service.val()

  checkItems(serviceType !== 1)
}

function checkItems(tokenAuth) {
  let enableItems
  if (tokenAuth) {
    enableItems = $token.val() !== ''
  } else {
    enableItems = $user.val() !== '' || $pass.val() !== ''
  }
  if (enableItems && $service.val() > 1) {
    if ($mirror.is(':checked')) {
      $items.not('[name="wiki"]').attr('disabled', true)
      $items.filter('[name="wiki"]').attr('disabled', false)
      return
    }
    $items.attr('disabled', false)
  } else {
    $items.attr('disabled', true)
  }
}
