import React, { Component } from 'react';
import Axios from 'axios';
import SuccessAlert from './SuccessAlert';
import ErrorAlert from './ErrorAlert';

export default class Add extends Component {
    constructor(){
        super();
        this.onChangeCode = this.onChangeCode.bind(this);
		this.onChangeTitle = this.onChangeTitle.bind(this);
		this.onChangeDescription = this.onChangeDescription.bind(this);
		this.onChangeWriter = this.onChangeWriter.bind(this);
		this.onChangeReleaseDate = this.onChangeReleaseDate.bind(this);
		this.onChangeIsReleased = this.onChangeIsReleased.bind(this);
        this.onSubmit = this.onSubmit.bind(this);

        this.state = {
            code:'',
			title:'',
			description:'',
			writer:'',
			release_date:'',
			is_released:'',
        }
    }

    onChangeCode(e){
		this.setState({
			code:e.target.value
		});
	}
	onChangeTitle(e){
		this.setState({
			title:e.target.value
		});
	}
	onChangeDescription(e){
		this.setState({
			description:e.target.value
		});
	}
	onChangeWriter(e){
		this.setState({
			writer:e.target.value
		});
	}
	onChangeReleaseDate(e){
		this.setState({
			release_date:e.target.value
		});
	}
	onChangeIsReleased(e){
		this.setState({
			is_released:e.target.value
		});
	}

    onSubmit(e){
        e.preventDefault();
        const fields = {
            code: this.state.code,
			title: this.state.title,
			description: this.state.description,
			writer: this.state.writer,
			release_date: this.state.release_date,
			is_released: this.state.is_released,
        }

        Axios.post('http://localhost:8000/api/kategori/store', fields)
        .then(res=>{
            this.setState({alert_message:"success"});
        }).catch(error=>{
            this.setState({alert_message:"error"});
        });
    }

    render() {
        return (
            <div>
            <hr/>

            {this.state.alert_message=="success"?<SuccessAlert/>:null}
            {this.state.alert_message=="error"?<ErrorAlert/>:null}

            <form onSubmit={this.onSubmit}>
                <div class="form-group">
                    <label for="title">title</label>
                    <input 
                        type="text" 
                        name="title" 
                        value="title"
                        value={this.state.title}
                        onChange={this.onChangeTitle} 
                    />
                </div>
                <div class="form-group">
                    <label for="description">description</label>
                    <input 
                        type="text" 
                        name="description" 
                        value="description"
                        value={this.state.description}
                        onChange={this.onChangeDescription} 
                    />
                </div>
                <div class="form-group">
                    <label for="writer">writer</label>
                    <input 
                        type="text" 
                        name="writer" 
                        value="writer"
                        value={this.state.writer}
                        onChange={this.onChangeWriter} 
                    />
                </div>
                <button type="submit" className="btn btn-primary">Submit</button>
                </form>
            </div>
        );
    }
}
