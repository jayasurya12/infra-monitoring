import React, { useState, useEffect } from 'react';
import axios from 'axios';

function App() {
  const [data, setData] = useState([]);

  useEffect(() => {
    axios.get('/api/systeminfo')
      .then(response => {
        setData(response.data);
      })
      .catch(error => {
        console.error('Error fetching data:', error);
      });
  }, []);

  return (
    <div>
      <h1>System Information</h1>
      <ul>
        {data.map((item, index) => (
          <li key={index}>{item.info}</li>
        ))}
      </ul>
    </div>
  );
}

export default App;
