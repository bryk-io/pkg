const fs = require('fs')
const jsonldChecker = require('jsonld-checker');

async function validateFile() {
  try {
    const data = fs.readFileSync(process.argv[2], 'utf8')
    const result = await jsonldChecker.check(data);
    console.log(JSON.stringify(result));
  } catch (err) {
    console.error(err)
  }
}

validateFile();
