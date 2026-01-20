// Test electron require from project directory
console.log('Running from:', __dirname);
console.log('process.versions.electron:', process.versions.electron);

const electron = require('electron');
console.log('typeof electron:', typeof electron);
console.log('electron value:', electron);

if (typeof electron === 'object') {
  console.log('electron.app:', electron.app);
  console.log('SUCCESS: electron module loaded correctly!');
  electron.app.whenReady().then(() => {
    console.log('Electron app is ready!');
    electron.app.quit();
  });
} else {
  console.log('FAIL: electron is not an object, got:', typeof electron);
  process.exit(1);
}
